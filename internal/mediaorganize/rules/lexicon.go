package rules

import (
	"strings"
	"sync/atomic"
)

const (
	DefaultMediaExtensions    = "mkv;mp4;avi;ts;mov;wmv;iso;m2ts;rmvb;flv;m4v;webm"
	DefaultMetadataExtensions = "nfo;ass;ssa;srt;sub;idx;sup;vtt;jpg;jpeg;png;webp;bmp"
	MaxFilenameBytes          = 235
	MaxReasonableSeason       = 99
)

var MediaTagFields = []string{
	"screen_size", "frame_rate", "video_codec", "audio_codec", "audio_channels",
}

var DefaultMediaTagOrder = []string{
	"screen_size", "frame_rate", "video_codec", "audio_codec", "audio_channels",
}

var GenericMediaDirNames = map[string]struct{}{
	"电影": {}, "影片": {}, "movie": {}, "movies": {},
	"电视剧": {}, "剧集": {}, "连续剧": {}, "tv": {}, "tv shows": {}, "shows": {}, "series": {},
	"动漫": {}, "动画": {}, "anime": {}, "media": {}, "video": {}, "videos": {}, "视频": {},
}

var KnownReleaseGroups = map[string]struct{}{
	"CHD": {}, "CHDBits": {}, "CHDTV": {}, "CHDWEB": {}, "CHDPAD": {}, "CHDHKTV": {},
	"WiKi": {}, "MTeam": {}, "MTeamTV": {}, "ADE": {}, "ADWeb": {},
	"HDS": {}, "HDSky": {}, "HDSWEB": {}, "HDH": {}, "HDC": {}, "HDArea": {}, "HDChina": {}, "HDCTV": {},
	"NTb": {}, "NTG": {}, "NTROPiC": {}, "FraMeSToR": {},
	"TLF": {}, "TLFCD": {}, "TLFGROUP": {},
	"OurBits": {}, "OurTV": {}, "OurPanda": {}, "iHD": {}, "OPS": {},
	"PuTao": {}, "Pter": {}, "PTHome": {},
	"DDR": {}, "TJUPT": {}, "JOY": {}, "BMDru": {},
	"FRDS": {}, "GREENOTEA": {}, "DBD": {}, "DKB": {}, "PandaMoon": {},
	"NF": {}, "AMZN": {}, "ATV": {}, "DSNP": {}, "HMAX": {}, "MAX": {}, "iT": {},
	"Sicario": {}, "Telly": {}, "TEPES": {}, "BTN": {}, "TRD": {}, "Pandamonium": {},
	"BiliBili": {}, "ByRA": {}, "ByMQ": {}, "NowYS": {}, "QHstudIo": {}, "RARBG": {},
	"YTS": {}, "YIFY": {}, "EVO": {}, "GalaxyRG": {}, "MeGusta": {},
	"FFans": {}, "MNHD": {}, "MTeamWEB": {},
	// 中文压制组 / 民间组
	"CMCT": {}, "beAst": {}, "BeAst": {}, "CHDWiKi": {}, "SUM": {}, "CEE": {},
	// 动漫字幕组（ASCII 尾部形态）
	"SweetSub": {}, "LoliHouse": {}, "Nekomoe": {}, "MCE": {}, "HYSUB": {}, "KTXP": {}, "MingY": {}, "ANi": {},
	// MoviePilot releasegroup.py 常用组（2026-08 对齐）
	"BeiTai": {}, "CarPT": {}, "StBOX": {}, "OneHD": {}, "Lee": {}, "xiaopie": {},
	"FLTTH": {}, "Ao": {}, "PbK": {}, "MGs": {}, "iLoveHD": {}, "iLoveTV": {},
	"MPAD": {}, "MWeb": {}, "Zone": {}, "AQLJ": {}, "Yumi": {}, "cXcY": {},
	"EPiC": {}, "k9611": {}, "tudou": {}, "DReam": {}, "DBTV": {},
	"beAstTV": {}, "HHWEB": {}, "HTPT": {}, "PTer": {}, "PTerDIY": {}, "PTerTV": {},
	"PTH": {}, "PTHomeTV": {}, "SGXT": {}, "SGTV": {},
	"FROG": {}, "FROGE": {}, "FROGWeb": {}, "UBits": {}, "UBWEB": {}, "UBTV": {},
	"NHDWEB": {}, "NGB": {}, "DoA": {}, "ARiN": {}, "ExREn": {}, "TTG": {},
	"BeyondHD": {}, "Cfandora": {}, "CtrlHD": {}, "CMRG": {},
	"DON": {}, "FLUX": {}, "HONEyG": {}, "NoGroup": {}, "SMURF": {},
	"Taengoo": {}, "trollHD": {},
	// 流媒体 / WEB 压制常见组
	"SHARKWEB": {}, "PiGoNF": {}, "AilMWeb": {},
}

// ReleaseGroups 不可变的制作组表（内置 + 用户自定义），构建后只读，可无锁并发读取。
type ReleaseGroups struct {
	table map[string]struct{}
}

// 当前生效的制作组表。写入方先完整构建新表再整体交换（atomic），
// 读取方永远看到完整一致的快照，不存在"半空表"窗口，也无 map 并发读写竞争。
var releaseGroupsPtr atomic.Pointer[ReleaseGroups]

var resolutionLikeNumbers = map[int]struct{}{
	360: {}, 480: {}, 540: {}, 576: {}, 720: {}, 1080: {}, 1440: {}, 2160: {}, 4320: {},
}

func init() {
	releaseGroupsPtr.Store(buildReleaseGroups(nil))
}

func buildReleaseGroups(userGroups []string) *ReleaseGroups {
	t := make(map[string]struct{}, len(KnownReleaseGroups)+len(userGroups))
	for g := range KnownReleaseGroups {
		t[toLowerASCII(g)] = struct{}{}
	}
	for _, g := range userGroups {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		t[toLowerASCII(g)] = struct{}{}
	}
	return &ReleaseGroups{table: t}
}

// ConfigureUserGroups 设置用户自定义制作组（逗号分隔输入框），
// 与内置表一起参与识别/剥离。传入 nil 或空串只清空用户组。
// 实现：先构建完整新表再整体原子交换，读取方无锁、无竞争。
func ConfigureUserGroups(groups []string) {
	releaseGroupsPtr.Store(buildReleaseGroups(groups))
}

func isKnownReleaseGroup(s string) bool {
	g := releaseGroupsPtr.Load()
	if g == nil {
		return false
	}
	_, ok := g.table[toLowerASCII(s)]
	return ok
}
