// 5성 획득 기록 인터페이스
export interface FiveStarRecord {
  name: string;
  time: string;
  pity: number;
  isPickUp: boolean;
}

// 개별 가챠 획득 기록 인터페이스
export interface Record {
  cardPoolType: string;
  resourceId: number;
  qualityLevel: number;
  resourceType: string;
  name: string;
  count: number;
  time: string;
}

// 가챠 통계 정보 인터페이스
export interface Stats {
  gachaType: number;
  gachaName: string;
  totalPulls: number;
  currentPity5: number;
  currentPity4: number;
  baseRate: number;
  expectedPulls: number;
  avgPulls: number;
  actualRate: number;
  luckScore: number;
  fiveStars: FiveStarRecord[] | null;
  records: Record[] | null;
  hasFiveStar: boolean;
}

// 운 점수 스타일 임계치 인터페이스
export type LuckScoreState = "worst" | "bad" | "normal" | "good" | "best";

export interface LuckScoreThreshold {
  minScore: number;
  state: LuckScoreState;
}
