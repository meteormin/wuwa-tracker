package tracker

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
	TotalPulls   int
	CurrentPity5 int
	CurrentPity4 int
	FiveStars    []FiveStarRecord
}
