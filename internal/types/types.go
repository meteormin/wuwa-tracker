package types

// Payload 는 API 요청 시 사용되는 구조체입니다.
type Payload struct {
	PlayerID     string `json:"playerId"`
	ServerID     string `json:"serverId"`
	LanguageCode string `json:"languageCode"`
	RecordID     string `json:"recordId"`
	CardPoolID   string `json:"cardPoolId"`
	CardPoolType int    `json:"cardPoolType"`
}

// FetchResult 는 FetchAllRecords의 전체 반환 결과를 담는 구조체입니다.
type FetchResult struct {
	Payload Payload             `json:"payload"`
	Records map[string][]Record `json:"records"`
}

// Record 는 단일 가챠 획득 기록을 나타냅니다.
type Record struct {
	CardPoolType string `json:"cardPoolType"`
	ResourceID   int    `json:"resourceId"`
	QualityLevel int    `json:"qualityLevel"`
	ResourceType string `json:"resourceType"`
	Name         string `json:"name"`
	Count        int    `json:"count"`
	Time         string `json:"time"`
}

// GachaResponse 는 API의 응답 전체를 감싸는 구조체입니다.
type GachaResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    []Record `json:"data"`
}

// LocaleData 는 로컬라이제이션 JSON 파일의 형식을 나타냅니다.
type LocaleData struct {
	SelectList map[string]string `json:"selectList"`
}

// FiveStarRecord 는 5성 획득 시점의 상세 정보를 저장합니다.
type FiveStarRecord struct {
	Name     string `json:"name"`
	Time     string `json:"time"`
	Pity     int    `json:"pity"`
	IsPickUp bool   `json:"isPickUp"` // 픽업 캐릭터/무기인지 여부 (픽뚫이면 false)
}

type GachaType struct {
	// ID GachaType Code
	ID int `json:"id"`
	// Key GachaType Key for display text
	Key string `json:"key"`
	// HasOffBannerDrop : True이면 5성 천장 초기화 여부를 판단해야함 (이중천장 대상)
	HasOffBannerDrop bool    `json:"hasOffBannerDrop"`
	Name             string  `json:"name"`
	BaseRate         float64 `json:"baseRate"`
	ExpectedPulls    int     `json:"expectedPulls"`
}

type GachaTypes struct {
	Items []GachaType `json:"items"`
}

func (gtypes *GachaTypes) MapFromLocaleData(localeData LocaleData) {
	// 현재는 select list만 필요하지만
	// 혹시 localeData에서 다른 필드들이 필요하게 되면 사용하기 위해
	selectList := localeData.SelectList
	for k, v := range selectList {
		for i, gt := range gtypes.Items {
			if gt.Key == k {
				gtypes.Items[i].Name = v
			}
		}
	}
}

// Stats 는 특정 가챠 배너에 대한 통계 지표입니다.
type Stats struct {
	GachaType     int              `json:"gachaType"`
	GachaName     string           `json:"gachaName"`
	TotalPulls    int              `json:"totalPulls"`
	CurrentPity5  int              `json:"currentPity5"`
	CurrentPity4  int              `json:"currentPity4"`
	BaseRate      float64          `json:"baseRate"`
	ExpectedPulls int              `json:"expectedPulls"`
	FiveStars     []FiveStarRecord `json:"fiveStars"`
	Records       []Record         `json:"records"`
	AvgPulls      float64          `json:"avgPulls"`
	ActualRate    float64          `json:"actualRate"`
	LuckScore     float64          `json:"luckScore"`
	HasFiveStar   bool             `json:"hasFiveStar"`
}

type ReportData struct {
	PlayerID string  `json:"playerId"`
	Stats    []Stats `json:"stats"`
}

type StandardFiveStarResource struct {
	Name       string `json:"name"`
	ResourceID int    `json:"resourceId"`
}

type StandardFiveStarResources struct {
	Items []StandardFiveStarResource `json:"items"`
}

func (s *StandardFiveStarResources) Contains(resourceId int) bool {
	for _, r := range s.Items {
		if r.ResourceID == resourceId {
			return true
		}
	}
	return false
}

type LuckScoreThreshold struct {
	MinScore float64 `json:"minScore"`
	State    string  `json:"state"`
}

// TrackRequest 는 가챠 기록 조회를 위한 URL 입력 요청 데이터 구조체입니다.
type TrackRequest struct {
	URL string `json:"url"`
}

// UploadRequest 는 JSON 로그 데이터를 직접 업로드하기 위한 구조체입니다.
type UploadRequest struct {
	FetchResult
}

// StatsResponse 는 프론트엔드로 반환될 표준 통계 응답 데이터 구조체입니다.
type StatsResponse struct {
	Success  bool    `json:"success"`
	PlayerID string  `json:"playerId"`
	Stats    []Stats `json:"stats"`
}

// ErrorResponse 는 다국어 지원을 위한 에러 응답 구조체입니다.
type ErrorResponse struct {
	Success  bool   `json:"success"`
	Error    string `json:"error"`
	ErrorKey string `json:"errorKey"`
}
