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
	Payload Payload              `json:"payload"`
	Records map[string][]Record  `json:"records"`
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

// Stats 는 특정 가챠 배너에 대한 통계 지표입니다.
type Stats struct {
	GachaType     int              `json:"gachaType"`
	GachaName     string           `json:"gachaName"`
	TotalPulls    int              `json:"totalPulls"`
	CurrentPity5  int              `json:"currentPity5"`
	CurrentPity4  int              `json:"currentPity4"`
	BaseRate      float64          `json:"baseRate"`
	ExpectedPulls float64          `json:"expectedPulls"`
	FiveStars     []FiveStarRecord `json:"fiveStars"`
	Records       []Record         `json:"records"`
	AvgPulls      float64          `json:"avgPulls"`
	ActualRate    float64          `json:"actualRate"`
	LuckScore     float64          `json:"luckScore"`
	HasFiveStar   bool
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
	ExpectedPulls    float64 `json:"expectedPulls"`
}

type GachaTypes struct {
	Items []GachaType `json:"items"`
}

func (gtypes *GachaTypes) MapFromSelectList(selectList map[string]string) {
	for k, v := range selectList {
		for i, gt := range gtypes.Items {
			if gt.Key == k {
				gtypes.Items[i].Name = v
			}
		}
	}
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
	MinScore   float64 `json:"minScore"`
	State      string  `json:"state"`
	ColorClass string  `json:"colorClass"`
	BgClass    string  `json:"bgClass"`
}
