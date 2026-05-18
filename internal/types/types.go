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

// =================== 내부 로직용 구조체 ===================

// FiveStarRecord 는 5성 획득 시점의 상세 정보를 저장합니다.
type FiveStarRecord struct {
	Name     string
	Time     string
	Pity     int
	IsPickUp bool // 픽업 캐릭터/무기인지 여부 (픽뚫이면 false)
}

// Stats 는 특정 가챠 배너에 대한 통계 지표입니다.
type Stats struct {
	GachaType    int
	GachaName    string
	TotalPulls   int
	CurrentPity5 int
	CurrentPity4 int
	FiveStars    []FiveStarRecord
	Records      []Record
}

type GachaType struct {
	// ID GachaType Code
	ID int
	// Key GachaType Key for display text
	Key string
	// HasOffBannerDrop : True이면 5성 천장 초기화 여부를 판단해야함 (이중천장 대상)
	HasOffBannerDrop bool
	Name             string
}

type GachaTypes struct {
	Items []GachaType
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
	Name       string
	ResourceID int
}

type StandardFiveStarResources struct {
	Items []StandardFiveStarResource
}

func (s *StandardFiveStarResources) Contains(resourceId int) bool {
	for _, r := range s.Items {
		if r.ResourceID == resourceId {
			return true
		}
	}
	return false
}
