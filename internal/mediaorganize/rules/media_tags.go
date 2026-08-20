package rules

import (
	"fmt"
	"regexp"
	"strings"
)

var qualityTokenRe = regexp.MustCompile(`(?i)(?:4320[pP]|2160[pP]|1080[pP]|720[pP]|480[pP]|4[Kk]|2[Kk]|8[Kk]|UHD|FHD|FullHD|WEB[-. ]?DL|WEB[-. ]?Rip|BluRay|BDRip|BDMV|BD25|BD50|HDTV|HDTVrip|DVDRip|DVD[-. ]?9|DVD[-. ]?5|REMUX|Repack|Proper|Extended|Director'?s[. ]Cut|Theatrical|Uncut|HDR10\+?|HDR|Dolby[. ]Vision|DoVi|SDR|HLG|10[. ]?bit|8[. ]?bit|H\.?264|H\.?265|HEVC|AVC|x264|x265|VP9|AV1|DTS[-.]?HD[. ]?MA|DTS[-.]?HD[. ]?HRA|DTS[-.]?HD|DTS[-.]?X|DTS|DDP|DD\+|DD|AC3|EAC3|TrueHD|Atmos|FLAC|AAC|OPUS|MP3|PCM|\d{2,3}(?:\.\d+)?fps|MultiAudio|Multi[. ]?Lang)`)

var (
	combinedAACChannelsRe  = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])AAC(\d\.\d)(?:$|[^A-Za-z0-9])`)
	combinedPCMChannelsRe  = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])PCM(\d\.\d)(?:$|[^A-Za-z0-9])`)
	combinedDDChannelsRe   = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])DD\+?(\d\.\d)(?:$|[^A-Za-z0-9])`)
	combinedAC3ChannelsRe  = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])AC3[-.]?(\d\.\d)(?:$|[^A-Za-z0-9])`)
	dtsXPatternRe          = regexp.MustCompile(`(?i)DTS[-.]?X`)
	dtsHDMAPatternRe       = regexp.MustCompile(`(?i)DTS[-.]?HD[-.]?MA`)
	dtsHDHRAPatternRe      = regexp.MustCompile(`(?i)DTS[-.]?HD[-.]?HRA`)
	dolbyTrueHDPatternRe   = regexp.MustCompile(`(?i)(?:Dolby[-.]?)?TrueHD`)
	channelLayoutRe        = regexp.MustCompile(`(?:^|[^0-9])([157]\.[01]|2\.0|2\.1|6\.1|1\.0)(?:$|[^0-9])`)
	frameRateTokenRe       = regexp.MustCompile(`(?i)(?:^|[^0-9])(\d{2,3}(?:\.\d+)?)\s*fps(?:$|[^0-9])`)
	bracketInnerRe         = regexp.MustCompile(`\[([^\]]+)\]|【([^】]+)】`)
)

var validChannelLayouts = map[string]struct{}{
	"1.0": {}, "2.0": {}, "2.1": {}, "4.0": {}, "5.0": {}, "5.1": {}, "6.1": {}, "7.0": {}, "7.1": {},
}

func EnrichMediaTagsFromFilename(name string, m map[string]any) {
	if m == nil {
		return
	}
	stem, _ := splitStemExt(strings.TrimSpace(name))
	if stem == "" {
		return
	}
	scanned := scanMediaTagsFromStem(stem)
	mergeScannedScreenSize(m, scanned.screenSize)
	mergeScannedString(m, "frame_rate", scanned.frameRate)
	mergeScannedString(m, "video_codec", scanned.videoCodec)
	mergeScannedString(m, "video_bit", scanned.videoBit)
	mergeScannedAudioCodec(m, scanned.audioCodec)
	mergeScannedString(m, "audio_channels", scanned.audioChannels)
	mergeScannedString(m, "audio_effect", scanned.audioEffect)
	mergeScannedString(m, "web_source", scanned.webSource)
	mergeScannedString(m, "edition", scanned.effect)
}

func mergeScannedScreenSize(m map[string]any, scanned string) {
	if scanned == "" {
		return
	}
	if mediaTagEmpty(m, "screen_size") {
		m["screen_size"] = scanned
		return
	}
	old := strings.ToLower(fmt.Sprint(m["screen_size"]))
	if screenSizeRank(scanned) > screenSizeRank(old) {
		m["screen_size"] = scanned
	}
}

func mergeScannedString(m map[string]any, key, scanned string) {
	if scanned == "" {
		return
	}
	if mediaTagEmpty(m, key) {
		m[key] = scanned
	}
}

func mergeScannedAudioCodec(m map[string]any, scanned string) {
	if scanned == "" {
		return
	}
	if mediaTagEmpty(m, "audio_codec") {
		m["audio_codec"] = scanned
		return
	}
	old := fmt.Sprint(m["audio_codec"])
	if audioCodecRank(scanned) > audioCodecRank(old) {
		m["audio_codec"] = scanned
	}
}

func audioCodecRank(codec string) int {
	_, rank := classifyAudioCodecToken(strings.TrimSpace(codec))
	if rank > 0 {
		return rank
	}
	switch strings.ToUpper(strings.TrimSpace(codec)) {
	case "DTS:X", "DTS-X":
		return 75
	case "DTS-HD MA":
		return 90
	case "DTS-HD HRA":
		return 85
	case "DTS-HD":
		return 80
	case "DTS":
		return 70
	case "TRUEHD":
		return 100
	case "DDP":
		return 60
	case "DD":
		return 55
	case "FLAC":
		return 50
	case "AAC":
		return 40
	case "PCM":
		return 30
	default:
		return 10
	}
}

func enrichParsedMediaTags(name string, p ParsedMedia) ParsedMedia {
	m := p.ToMap()
	EnrichMediaTagsFromFilename(name, m)
	// 统一视频编码形态（guessit/本地扫描可能给出 H.265/HEVC/x265 等不同写法）
	if raw, ok := m["video_codec"].(string); ok && raw != "" {
		if norm := NormalizeVideoCodec(raw); norm != "" {
			m["video_codec"] = norm
		}
	}
	return parsedFromMap(m)
}

func mediaTagEmpty(m map[string]any, key string) bool {
	if m == nil {
		return true
	}
	v, ok := m[key]
	if !ok || v == nil {
		return true
	}
	return strings.TrimSpace(fmt.Sprint(v)) == ""
}

type mediaTagScanResult struct {
	screenSize    string
	frameRate     string
	videoCodec    string
	audioCodec    string
	audioChannels string
	audioEffect   string // Atmos 等音频特效
	videoBit      string
	webSource     string // 流媒体平台（NF->Netflix 等）
	effect        string // 视频特效（HDR10+/DoVi/Dolby Vision 等，MoviePilot: resource_effect）
}

func scanMediaTagsFromStem(stem string) mediaTagScanResult {
	out := mediaTagScanResult{}
	scanText := expandStemForTagScan(stem)

	applyCombinedPatterns(scanText, &out)

	screenRank := -1
	audioRank := -1
	for _, token := range qualityTokenRe.FindAllString(scanText, -1) {
		field, value, rank := classifyQualityToken(token)
		if field == "" || value == "" {
			continue
		}
		switch field {
		case "screen_size":
			if rank > screenRank {
				screenRank = rank
				out.screenSize = value
			}
		case "frame_rate":
			if out.frameRate == "" {
				out.frameRate = value
			}
		case "video_codec":
			if out.videoCodec == "" {
				out.videoCodec = value
			}
		case "video_bit":
			if out.videoBit == "" {
				out.videoBit = value
			}
		case "effect":
			if out.effect == "" {
				out.effect = value
			}
		case "audio_codec":
			if rank > audioRank {
				audioRank = rank
				out.audioCodec = value
			}
		case "audio_channels":
			if out.audioChannels == "" {
				out.audioChannels = value
			}
		}
	}
	// Atmos 等音频特效（qualityTokenRe 已含 Atmos，但 classifyAudioCodecToken 会忽略）
	if out.audioEffect == "" {
		if m := atmosTokenRe.FindStringSubmatch(scanText); len(m) >= 2 {
			out.audioEffect = "Atmos"
		}
	}
	// 流媒体平台（MoviePilot: 平台简称出现在 WEB-DL/WEBRip 附近才记）
	if out.webSource == "" {
		out.webSource = detectStreamingPlatform(scanText)
	}
	return out
}

// atmosTokenRe 匹配 Atmos / Dolby Atmos 音频特效
var atmosTokenRe = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])(?:Dolby[. ]?)?(Atmos)(?:$|[^A-Za-z0-9])`)

// streamingPlatforms 流媒体平台简称表（MoviePilot streamingplatform.py 常用项）
var streamingPlatforms = map[string]string{
		"10Play": "10Play",
		"3SAT": "3sat",
		"3sat": "3sat",
		"7PLUS": "7plus",
		"7plus": "7plus",
		"9NOW": "9Now",
		"9Now": "9Now",
		"A&E": "A&E",
		"A3P": "Atresplayer",
		"ABMA": "Abema",
		"AE": "A&E",
		"AHA": "aha",
		"ALL4": "Channel 4",
		"AMC": "AMC",
		"AMZN": "Amazon",
		"ANPL": "Animal Planet",
		"AO": "AnimeOnegai",
		"AOD": "Anime on Demand",
		"APPS": "Disney+ MENA",
		"ARD": "ARD",
		"ARGP": "Argo",
		"ARTE": "Arte",
		"AS": "Adult Swim",
		"ATVP": "Apple TV+",
		"Abema": "Abema",
		"AbemaTV": "Abema",
		"Adult Swim": "Adult Swim",
		"Amazon": "Amazon",
		"Animal Planet": "Animal Planet",
		"Anime on Demand": "Anime on Demand",
		"AnimeOnegai": "AnimeOnegai",
		"Apple TV+": "Apple TV+",
		"Argo": "Argo",
		"Arte": "Arte",
		"Atresplayer": "Atresplayer",
		"B-Global": "B-Global",
		"BBC": "BBC",
		"BBC iPlayer": "BBC iPlayer",
		"BCH": "Bandai Channel",
		"BCORE": "Bravia Core",
		"BG": "B-Global",
		"BK": "Bentkey",
		"BNGE": "Binge",
		"BOOM": "Boomerang",
		"BRAV": "BravoTV",
		"BRTB": "Brtb TV",
		"BYU": "BYUtv",
		"BYUtv": "BYUtv",
		"Bandai Channel": "Bandai Channel",
		"Bentkey": "Bentkey",
		"Binge": "Binge",
		"Boomerang": "Boomerang",
		"Bravia Core": "Bravia Core",
		"BravoTV": "BravoTV",
		"BritBox": "BritBox",
		"Brtb TV": "Brtb TV",
		"C More": "C More",
		"CATCHPLAY": "CATCHPLAY+",
		"CATCHPLAY+": "CATCHPLAY+",
		"CBC": "CBC Gem",
		"CBC Gem": "CBC Gem",
		"CBS": "CBS",
		"CC": "Comedy Central",
		"CLBI": "Club illico",
		"CMAX": "Cinemax",
		"CMOR": "C More",
		"CNBC": "CNBC",
		"CNLP": "Canal+",
		"CNN+": "CNN+",
		"CNNP": "CNN+",
		"COOK": "Cooking Channel",
		"CPE": "Cineplex Entertainment",
		"CPP": "CATCHPLAY+",
		"CR": "Crunchyroll",
		"CRAV": "Crave",
		"CRIT": "Criterion Channel",
		"CRKL": "Crackle",
		"CTHP": "CATCHPLAY+",
		"CTV": "CTV",
		"CUR": "Curiosity Stream",
		"CW": "The CW",
		"CW Seed": "CW Seed",
		"CWS": "CW Seed",
		"Canal+": "Canal+",
		"Channel 4": "Channel 4",
		"Channel 5": "Channel 5",
		"Cinemax": "Cinemax",
		"Cineplex Entertainment": "Cineplex Entertainment",
		"Cineverse": "Cineverse",
		"Club illico": "Club illico",
		"Comedy Central": "Comedy Central",
		"Cooking Channel": "Cooking Channel",
		"Crackle": "Crackle",
		"Crave": "Crave",
		"Criterion Channel": "Criterion Channel",
		"Crunchyroll": "Crunchyroll",
		"Curiosity Stream": "Curiosity Stream",
		"DANET": "DANET",
		"DANT": "DANET",
		"DC Universe": "DC Universe",
		"DCU": "DC Universe",
		"DDY": "Digiturk Dilediğin Yerde",
		"DEST": "Destination America",
		"DISC": "Discovery Channel",
		"DLWP": "DailyWire+",
		"DMM": "DMM",
		"DPLY": "dplay",
		"DRPO": "Dropout",
		"DSCP": "Discovery+",
		"DSNP": "Disney+",
		"DSNY": "Disney Networks",
		"DW": "DailyWire+",
		"DailyWire+": "DailyWire+",
		"Dekkoo": "Dekkoo",
		"Destination America": "Destination America",
		"Digiturk Dilediğin Yerde": "Digiturk Dilediğin Yerde",
		"Discovery Channel": "Discovery Channel",
		"Discovery Velocity": "Discovery Velocity",
		"Discovery+": "Discovery+",
		"Disney Networks": "Disney Networks",
		"Disney+": "Disney+",
		"Disney+ MENA": "Disney+ MENA",
		"Dropout": "Dropout",
		"E!": "E!",
		"EPIX": "EPIX MGM+",
		"EPIX MGM+": "EPIX MGM+",
		"ESQ": "Esquire",
		"ETV": "E!",
		"Esquire": "Esquire",
		"FAA": "Filmarchiv Austria",
		"FANDOR": "fandor",
		"FAWESOME": "Fawesome",
		"FBWatch": "Facebook Watch",
		"FILMIN": "Filmin",
		"FILMINGO": "filmingo",
		"FILMZIE": "Filmzie",
		"FOOD": "Food Network",
		"FPT": "FPT Play",
		"FPT Play": "FPT Play",
		"FPTP": "FPT Play",
		"FREE": "Freeform",
		"FTV": "France.tv",
		"FUBO": "fuboTV",
		"FUNi": "Funimation",
		"FXTL": "Foxtel Now",
		"FYI": "FYI Network",
		"FYI Network": "FYI Network",
		"Facebook Watch": "Facebook Watch",
		"Fawesome": "Fawesome",
		"Filmarchiv Austria": "Filmarchiv Austria",
		"Filmin": "Filmin",
		"Filmzie": "Filmzie",
		"FlixLatino": "FlixLatino",
		"FlixOlé": "FlixOlé",
		"Flixole": "FlixOlé",
		"Food Network": "Food Network",
		"Foxtel Now": "Foxtel Now",
		"France.tv": "France.tv",
		"Freeform": "Freeform",
		"Funimation": "Funimation",
		"GAIA": "Gaia",
		"GLBO": "Globoplay",
		"GLOB": "GloboSat Play",
		"GO90": "go90",
		"Gaga": "GagaOOLala",
		"GagaOOLala": "GagaOOLala",
		"Gaia": "Gaia",
		"GloboSat Play": "GloboSat Play",
		"Globoplay": "Globoplay",
		"Go3": "Go3",
		"Google Play": "Google Play",
		"HBO": "HBO",
		"HBO GO": "HBO GO",
		"HBOGO": "HBO GO",
		"HGTV": "HGTV",
		"HIDI": "HIDIVE",
		"HIDIVE": "HIDIVE",
		"HIST": "History Channel",
		"HLMK": "Hallmark",
		"HMAX": "Max",
		"HPLAY": "Hungama Play",
		"HS": "Hotstar",
		"HULU": "Hulu Networks",
		"Hallmark": "Hallmark",
		"Hami": "Hami Video",
		"Hami Video": "Hami Video",
		"HamiVideo": "Hami Video",
		"History Channel": "History Channel",
		"HoiChoi": "Hoichoi",
		"Hoichoi": "Hoichoi",
		"Hotstar": "Hotstar",
		"Hulu Networks": "Hulu Networks",
		"HuluJP": "Hulu Networks",
		"Hungama Play": "Hungama Play",
		"ITV": "ITV",
		"ITVX": "ITV",
		"IVI": "Ivi",
		"Ici TOU.TV": "Ici TOU.TV",
		"IndieFlix": "IndieFlix",
		"Ivi": "Ivi",
		"JC": "JioCinema",
		"JONU": "Jonu Play",
		"JOYN": "Joyn",
		"JioCinema": "JioCinema",
		"Jonu Play": "Jonu Play",
		"Joyn": "Joyn",
		"KKTV": "KKTV",
		"KLASSIKI": "Klassiki",
		"KNOW": "Knowledge Network",
		"KNPY": "Kanopy",
		"KS": "Kaleidescape",
		"Kaleidescape": "Kaleidescape",
		"Kanopy": "Kanopy",
		"Klassiki": "Klassiki",
		"Knowledge Network": "Knowledge Network",
		"LACINETEK": "LaCinetek",
		"LFTL": "Laftel",
		"LFTLNET": "Laftel",
		"LGP": "Lionsgate Play",
		"LIFE": "Lifetime",
		"LINE TV": "LINE TV",
		"LINETV": "LINE TV",
		"LN": "Love Nature",
		"LOCIPO": "LOCIPO",
		"LaCinetek": "LaCinetek",
		"Laftel": "Laftel",
		"Lemino": "Lemino",
		"LiTV": "LiTV",
		"Lifetime": "Lifetime",
		"Lionsgate Play": "Lionsgate Play",
		"Love Nature": "Love Nature",
		"MA": "Movies Anywhere",
		"MBC": "MBC",
		"MBS": "MBS",
		"MMAX": "ManoramaMAX",
		"MNBC": "MSNBC",
		"MP": "Movistar Plus+",
		"MS": "Microsoft Store",
		"MSNBC": "MSNBC",
		"MTOD": "Motor Trend OnDemand",
		"MUBI": "Mubi",
		"MW": "meWATCH",
		"MY5": "Channel 5",
		"ManoramaMAX": "ManoramaMAX",
		"Max": "Max",
		"Maxdome": "Maxdome",
		"Microsoft Store": "Microsoft Store",
		"Mitele": "Mitele",
		"Motor Trend OnDemand": "Motor Trend OnDemand",
		"Movies Anywhere": "Movies Anywhere",
		"Movistar Plus+": "Movistar Plus+",
		"Mubi": "Mubi",
		"MyTVS": "MyTVSuper",
		"MyTVSuper": "MyTVSuper",
		"MyVideo": "MyVideo",
		"NATG": "National Geographic",
		"NBLA": "Nebula",
		"NF": "Netflix",
		"NICK": "Nickelodeon",
		"NOW": "Now",
		"NYMEY": "Nymey",
		"National Geographic": "National Geographic",
		"Nebula": "Nebula",
		"Netflix": "Netflix",
		"Nickelodeon": "Nickelodeon",
		"Now": "Now",
		"Now E": "Now E",
		"NowE": "Now E",
		"NowPlayer": "NowPlayer",
		"Nymey": "Nymey",
		"ODK": "OnDemandKorea",
		"OKKO": "Okko",
		"OSN": "OSN+",
		"OSN+": "OSN+",
		"OV": "OceanVeil",
		"OVID": "OVID",
		"OXGN": "Oxygen",
		"OceanVeil": "OceanVeil",
		"Okko": "Okko",
		"OnDemandKorea": "OnDemandKorea",
		"Oxygen": "Oxygen",
		"PBS": "PBS",
		"PBS KIDS": "PBS KIDS",
		"PBSK": "PBS KIDS",
		"PCOK": "Peacock",
		"PLAY": "Google Play",
		"PLEX": "Plex",
		"PMNT": "Paramount Network",
		"PMTP": "Paramount+",
		"POGO": "PokerGO",
		"PSN": "PlayStation Network",
		"PUHU": "puhutv",
		"Paramount Network": "Paramount Network",
		"Paramount+": "Paramount+",
		"Peacock": "Peacock",
		"PlayStation Network": "PlayStation Network",
		"Plex": "Plex",
		"Pluto TV": "Pluto TV",
		"PlutoTV": "Pluto TV",
		"PokerGO": "PokerGO",
		"QIBI": "Quibi",
		"Quibi": "Quibi",
		"REVEEL": "Reveel",
		"RKTN": "Rakuten TV",
		"ROKU": "Roku",
		"RTE": "RTÉ",
		"RTL": "RTL+",
		"RTL+": "RTL+",
		"RTÉ": "RTÉ",
		"RUNTIME": "Runtime",
		"Rakuten TV": "Rakuten TV",
		"Rakuten Viki": "Rakuten Viki",
		"Reveel": "Reveel",
		"Roku": "Roku",
		"Runtime": "Runtime",
		"SAINA": "Saina Play",
		"SAMANSA": "SAMANSA",
		"SBS": "SBS",
		"SESO": "Seeso",
		"SF": "SF Anytime",
		"SF Anytime": "SF Anytime",
		"SHAHID": "Shahid",
		"SHDR": "Shudder",
		"SHO": "Showtime",
		"SKST": "SkyShowtime",
		"SMNS": "SAMANSA",
		"SNXT": "Sun NXT",
		"SP": "Saina Play",
		"SPIK": "Spike",
		"SS": "Simply South",
		"STAN": "Stan",
		"STARZ": "STARZ",
		"STRP": "Star+",
		"STZ": "STARZ",
		"SVT": "Sveriges Television",
		"SYFY": "SyFy",
		"Saina Play": "Saina Play",
		"Seeso": "Seeso",
		"Shahid": "Shahid",
		"Showtime": "Showtime",
		"Shudder": "Shudder",
		"Simply South": "Simply South",
		"SkyShowtime": "SkyShowtime",
		"Spike": "Spike",
		"Stan": "Stan",
		"Star+": "Star+",
		"Sun NXT": "Sun NXT",
		"Sveriges Television": "Sveriges Television",
		"SyFy": "SyFy",
		"TCM": "TCM",
		"TEN": "10Play",
		"TENK": "Tënk",
		"TIMV": "TIMvision",
		"TIMvision": "TIMvision",
		"TK": "Tentkotta",
		"TLC": "TLC",
		"TNT": "TNT",
		"TOU": "Ici TOU.TV",
		"TROMA": "Troma",
		"TRVL": "Travel Channel",
		"TUBI": "TubiTV",
		"TV 2": "TV 2",
		"TV Land": "TV Land",
		"TV2": "TV 2",
		"TV4": "TV4",
		"TVER": "TVer",
		"TVING": "TVING",
		"TVL": "TV Land",
		"TVNZ": "TVNZ",
		"TVO": "tvo",
		"TVer": "TVer",
		"Tentkotta": "Tentkotta",
		"The CW": "The CW",
		"Travel Channel": "Travel Channel",
		"Troma": "Troma",
		"TubiTV": "TubiTV",
		"Tënk": "Tënk",
		"U-NEXT": "U-NEXT",
		"UKTV": "UKTV",
		"UNXT": "U-NEXT",
		"USA Network": "USA Network",
		"USAN": "USA Network",
		"VH1": "VH1",
		"VIAP": "Viaplay",
		"VICE": "Viceland",
		"VIDIO": "Vidio",
		"VIKI": "Rakuten Viki",
		"VIU": "Viu",
		"VLCT": "Discovery Velocity",
		"VMAX": "vivamax",
		"VMEO": "Vimeo",
		"VMJ": "VideoMarket",
		"VOYO": "Voyo",
		"VRV": "VRV Defunct",
		"VRV Defunct": "VRV Defunct",
		"ViX": "ViX",
		"Viaplay": "Viaplay",
		"Viceland": "Viceland",
		"VideoMarket": "VideoMarket",
		"Vidio": "Vidio",
		"Vimeo": "Vimeo",
		"Viu": "Viu",
		"Voyo": "Voyo",
		"WAKA": "Wakanim",
		"WAKANIM": "Wakanim",
		"WATCH IT": "WATCH IT",
		"WAVO": "WAVO",
		"WAVVE": "Wavve",
		"WOW": "WOW",
		"WOW Presents Plus": "WOW Presents Plus",
		"WOWP": "WOW Presents Plus",
		"WTCH": "Watcha",
		"WWE Network": "WWE Network",
		"WWEN": "WWE Network",
		"Wakanim": "Wakanim",
		"Watcha": "Watcha",
		"WatchiT": "WATCH IT",
		"Wavve": "Wavve",
		"WeTV": "WeTV",
		"YT": "YouTube",
		"YouTube": "YouTube",
		"ZDF": "ZDF",
		"ZEE5": "ZEE5",
		"aha": "aha",
		"dTV": "dTV",
		"dplay": "dplay",
		"fandor": "fandor",
		"filmingo": "filmingo",
		"friDay": "friDay",
		"fuboTV": "fuboTV",
		"go90": "go90",
		"iP": "BBC iPlayer",
		"iT": "iTunes",
		"iTunes": "iTunes",
		"meWATCH": "meWATCH",
		"ofiii": "ofiii",
		"puhutv": "puhutv",
		"tvo": "tvo",
		"vivamax": "vivamax",
}

// detectStreamingPlatform 扫描文本中的流媒体平台简称；仅在文本含 WEB 相关字样时返回（MoviePilot 语义）
func detectStreamingPlatform(text string) string {
	upper := strings.ToUpper(text)
	hasWeb := strings.Contains(upper, "WEB-DL") || strings.Contains(upper, "WEBDL") ||
		strings.Contains(upper, "WEBRIP") || strings.Contains(upper, "WEB.RIP") ||
		strings.Contains(upper, "WEB.DL")
	if !hasWeb {
		return ""
	}
	for abbr, name := range streamingPlatforms {
		// 平台简称作为独立 token 匹配（前后非字母数字）
		re := regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])` + regexp.QuoteMeta(abbr) + `(?:$|[^A-Za-z0-9])`)
		if re.MatchString(text) {
			return name
		}
	}
	return ""
}

func expandStemForTagScan(stem string) string {
	var b strings.Builder
	b.WriteString(stem)
	b.WriteByte(' ')
	for _, inner := range bracketInners(stem) {
		b.WriteString(strings.TrimSpace(inner))
		b.WriteByte(' ')
	}
	return b.String()
}

func applyCombinedPatterns(text string, out *mediaTagScanResult) {
	if m := combinedAACChannelsRe.FindStringSubmatch(text); len(m) >= 2 {
		out.audioCodec = "AAC"
		out.audioChannels = m[1]
	}
	if m := combinedPCMChannelsRe.FindStringSubmatch(text); len(m) >= 2 {
		out.audioCodec = "PCM"
		out.audioChannels = m[1]
	}
	if m := combinedDDChannelsRe.FindStringSubmatch(text); len(m) >= 2 {
		if out.audioCodec == "" {
			out.audioCodec = "DDP"
		}
		if out.audioChannels == "" {
			out.audioChannels = m[1]
		}
	}
	if m := combinedAC3ChannelsRe.FindStringSubmatch(text); len(m) >= 2 {
		if out.audioCodec == "" {
			out.audioCodec = "DD"
		}
		if out.audioChannels == "" {
			out.audioChannels = m[1]
		}
	}
	if dtsXPatternRe.MatchString(text) {
		out.audioCodec = "DTS:X"
	} else if dtsHDMAPatternRe.MatchString(text) && out.audioCodec == "" {
		out.audioCodec = "DTS-HD MA"
	}
	if dtsHDHRAPatternRe.MatchString(text) && out.audioCodec == "" {
		out.audioCodec = "DTS-HD HRA"
	}
	if dolbyTrueHDPatternRe.MatchString(text) && out.audioCodec == "" {
		out.audioCodec = "TrueHD"
	}
	if out.audioChannels == "" {
		if m := channelLayoutRe.FindStringSubmatch(text); len(m) >= 2 {
			out.audioChannels = m[1]
		}
	}
}

func classifyQualityToken(raw string) (field, value string, rank int) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return "", "", 0
	}

	if m := frameRateTokenRe.FindStringSubmatch(" " + token + " "); len(m) >= 2 {
		return "frame_rate", normalizeFrameRate(m[1]), 0
	}
	if strings.HasSuffix(strings.ToLower(token), "fps") {
		return "frame_rate", normalizeFrameRate(strings.TrimSuffix(strings.ToLower(token), "fps")), 0
	}

	if bit := classifyVideoBitToken(token); bit != "" {
		return "video_bit", bit, 0
	}
	if eff := classifyEffectToken(token); eff != "" {
		return "effect", eff, 0
	}
	if size, r := classifyScreenSizeToken(token); size != "" {
		return "screen_size", size, r
	}
	if ch := classifyChannelToken(token); ch != "" {
		return "audio_channels", ch, 0
	}
	if vc := classifyVideoCodecToken(token); vc != "" {
		return "video_codec", vc, 0
	}
	if ac, pr := classifyAudioCodecToken(token); ac != "" {
		return "audio_codec", ac, pr
	}
	return "", "", 0
}

func classifyScreenSizeToken(token string) (string, int) {
	switch strings.ToUpper(strings.TrimSpace(token)) {
	case "8K":
		return "4320p", 5
	case "4K", "UHD":
		return "2160p", 4
	case "2K":
		return "1080p", 3
	case "FHD", "FULLHD":
		return "1080p", 3
	}
	m := regexp.MustCompile(`(?i)^(4320|2160|1080|720|480)[pP]$`).FindStringSubmatch(token)
	if len(m) >= 2 {
		size := strings.ToLower(m[1]) + "p"
		return size, screenSizeRank(size)
	}
	return "", 0
}

func classifyChannelToken(token string) string {
	if m := regexp.MustCompile(`^(\d\.\d)$`).FindStringSubmatch(strings.TrimSpace(token)); len(m) >= 2 {
		if _, ok := validChannelLayouts[m[1]]; ok {
			return m[1]
		}
	}
	return ""
}

// classifyEffectToken 识别视频特效（MoviePilot: resource_effect，如 HDR10+/DoVi）
func classifyEffectToken(token string) string {
	norm := strings.ToLower(strings.TrimSpace(token))
	norm = strings.ReplaceAll(norm, " ", "")
	switch norm {
	case "hdr10+", "hdr10plus":
		return "HDR10+"
	case "hdr10", "hdr10p":
		return "HDR10"
	case "hdr":
		return "HDR"
	case "dolbyvision", "dovi", "dv":
		return "DoVi"
	case "sdr":
		return "SDR"
	case "hlg":
		return "HLG"
	case "3d":
		return "3D"
	default:
		return ""
	}
}

func classifyVideoBitToken(token string) string {
	norm := strings.ToLower(strings.TrimSpace(token))
	norm = strings.ReplaceAll(norm, " ", "")
	switch norm {
	case "10bit", "10-bit":
		return "10bit"
	case "8bit", "8-bit":
		return "8bit"
	case "12bit", "12-bit":
		return "12bit"
	default:
		return ""
	}
}

func classifyVideoCodecToken(token string) string {
	// 归一化：HEVC/H265/H.265 -> x265；AVC/H264/H.264 -> x264（统一形态，避免同编码多种写法导致重复文件）
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(token), ".", "")) {
	case "h265", "hevc", "x265":
		return "x265"
	case "h264", "avc", "x264":
		return "x264"
	case "av1":
		return "AV1"
	case "vp9":
		return "VP9"
	default:
		return ""
	}
}

// NormalizeVideoCodec 统一视频编码形态（guessit 等外部来源可能给 H.265/HEVC 等写法）
func NormalizeVideoCodec(codec string) string {
	return classifyVideoCodecToken(codec)
}

func classifyAudioCodecToken(token string) (string, int) {
	norm := strings.ToLower(strings.TrimSpace(token))
	norm = regexp.MustCompile(`[\s._-]+`).ReplaceAllString(norm, " ")
	switch {
	case strings.Contains(norm, "dts-x"), norm == "dts:x":
		return "DTS:X", 75
	case strings.Contains(norm, "dts-hd ma"), norm == "dtshdma":
		return "DTS-HD MA", 90
	case strings.Contains(norm, "dts-hd hra"):
		return "DTS-HD HRA", 85
	case strings.Contains(norm, "dts-hd"):
		return "DTS-HD", 80
	case norm == "dts":
		return "DTS", 70
	case norm == "ddp", norm == "dd+", strings.Contains(norm, "eac3"):
		return "DDP", 60
	case norm == "dd", norm == "ac3":
		return "DD", 55
	case strings.Contains(norm, "truehd"):
		return "TrueHD", 100
	case norm == "flac":
		return "FLAC", 50
	case norm == "aac":
		return "AAC", 40
	case norm == "pcm":
		return "PCM", 30
	case norm == "opus":
		return "Opus", 25
	case norm == "mp3":
		return "MP3", 20
	case norm == "atmos", strings.Contains(norm, "dolby atmos"):
		return "", 0
	default:
		return "", 0
	}
}

// classifyAudioEffectToken 识别音频特效（MoviePilot 里并入 audio_encode 尾部，如 "DDP 5.1 Atmos"）
func classifyAudioEffectToken(token string) string {
	norm := strings.ToLower(strings.TrimSpace(token))
	norm = strings.ReplaceAll(norm, " ", "")
	switch norm {
	case "atmos", "dolbyatmos":
		return "Atmos"
	default:
		return ""
	}
}

func bracketInners(stem string) []string {
	out := make([]string, 0)
	for _, m := range bracketInnerRe.FindAllStringSubmatch(stem, -1) {
		if len(m) >= 2 && strings.TrimSpace(m[1]) != "" {
			out = append(out, m[1])
		}
		if len(m) >= 3 && strings.TrimSpace(m[2]) != "" {
			out = append(out, m[2])
		}
	}
	return out
}

func screenSizeRank(token string) int {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "4320p", "8k":
		return 5
	case "2160p", "4k", "uhd":
		return 4
	case "1080p", "fhd", "2k":
		return 3
	case "720p":
		return 2
	case "480p":
		return 1
	default:
		return 0
	}
}
