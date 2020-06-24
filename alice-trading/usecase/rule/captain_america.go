package rule

import (
	"github.com/fmyaaaaaaa/Alice/alice-trading/domain"
	"github.com/fmyaaaaaaa/Alice/alice-trading/domain/enum"
	"github.com/fmyaaaaaaa/Alice/alice-trading/usecase"
	"log"
)

type CaptainAmerica struct {
	DB                   usecase.DBRepository
	TrendStatus          usecase.TrendStatusRepository
	CaptainAmericaStatus usecase.CaptainAmericaStatusRepository
	TradeRuleStatus      usecase.TradeRuleStatusRepository
}

// 足データをもとにセットアップを検証します。
func (c CaptainAmerica) JudgementSetup(lastCandle, currentCandle *domain.BidAskCandles, instrument string, granularity enum.Granularity) {
	// 同一銘柄、時間足で既に売買中またはセットアップ中の場合はセットアップ検証をスキップする。
	if ok := c.isExistSetupOrTrade(instrument, granularity); ok {
		return
	}
	setUp := false
	// 一つ前の線種と今回の線種を比較し、同一の場合はセットアップ
	if lastCandle.Line == currentCandle.Line {
		setUp = true
	}
	// セットアップ情報を更新する。
	if setUp {
		captainAmericaStatus := domain.NewCaptainAmericaStatus(instrument, granularity, currentCandle.Line, currentCandle.GetCloseMid(), true, false)
		c.CreateOrUpdateCaptainAmericaStatus(captainAmericaStatus)
		tradeRuleStatus := domain.NewTradeRuleStatus(enum.CaptainAmerica, instrument, granularity, currentCandle.Candles.Time)
		c.CreateOrUpdateTradeRuleStatus(tradeRuleStatus)
		log.Println("CaptainAmerica setup happened ", instrument, granularity)
	}
}

// 足データをもとにトレード計画を判定します。
func (c CaptainAmerica) JudgementTradePlan(tradeRuleStatus domain.TradeRuleStatus, currentCandle *domain.BidAskCandles, instrument string, granularity enum.Granularity) {
	// セットアップを取得
	captainAmericaStatus := c.GetCaptainAmericaStatus(instrument, granularity)
	tradePlan := false
	// TODO:資金管理から数量、トレーリングストップ値幅を取得し、OrderManagerから注文を実行する
	switch captainAmericaStatus.Line {
	case enum.Positive:
		if captainAmericaStatus.SetupPrice <= currentCandle.GetCloseMid() {
			tradePlan = true
			log.Println("CaptainAmerica trade happened", currentCandle.Candles.Time, instrument, granularity, currentCandle.GetCloseMid())
		}
	case enum.Negative:
		if captainAmericaStatus.SetupPrice >= currentCandle.GetCloseMid() {
			tradePlan = true
			log.Println("CaptainAmerica trade happened", currentCandle.Candles.Time, instrument, granularity, currentCandle.GetCloseMid())
		}
	}

	// トレード計画の結果に応じて、売買ルールの状態を変更する。
	c.HandleCaptainAmericaStatus(&captainAmericaStatus, tradePlan)
	if tradePlan || captainAmericaStatus.SecondJudge {
		c.CompleteTradeRuleStatus(&tradeRuleStatus)
	}
}

// 銘柄、足種でセットアップ済みまたは取引済みかどうかを確認します。
// セットアップ済み、取引済みの場合はtrueを返却します。
func (c CaptainAmerica) isExistSetupOrTrade(instrument string, granularity enum.Granularity) bool {
	captainAmericaStatus := c.GetCaptainAmericaStatus(instrument, granularity)
	if captainAmericaStatus.SetupStatus || captainAmericaStatus.TradeStatus {
		return true
	}
	return false
}

// 2回目のトレード計画検証対象が存在するかどうかを確認します。
// SecondJudge対象（true）のセットアップが存在する場合はtrueを返却します。
func (c CaptainAmerica) IsExistSecondJudgementTradePlan(instrument string, granularity enum.Granularity) bool {
	captainAmericaStatus := c.GetCaptainAmericaStatus(instrument, granularity)
	return captainAmericaStatus.SecondJudge
}

// CaptainAmericaStatusを取得します。
func (c CaptainAmerica) GetCaptainAmericaStatus(instrument string, granularity enum.Granularity) domain.CaptainAmericaStatus {
	DB := c.DB.Connect()
	return c.CaptainAmericaStatus.FindByInstrumentAndGranularity(DB, instrument, granularity)
}

// CaptainAmericaStatusを作成します。
// 既に同一銘柄、足種で作成済みの場合は更新します。
func (c CaptainAmerica) CreateOrUpdateCaptainAmericaStatus(captainAmericaStatus *domain.CaptainAmericaStatus) {
	DB := c.DB.Connect()
	if target := c.CaptainAmericaStatus.FindByInstrumentAndGranularity(DB, captainAmericaStatus.Instrument, captainAmericaStatus.Granularity); target.ID == 0 {
		c.CaptainAmericaStatus.Create(DB, captainAmericaStatus)
	} else {
		params := map[string]interface{}{
			"line":         captainAmericaStatus.Line,
			"setup_price":  captainAmericaStatus.SetupPrice,
			"setup_status": captainAmericaStatus.SetupStatus,
			"trade_status": captainAmericaStatus.TradeStatus,
		}
		c.CaptainAmericaStatus.Update(DB, &target, params)
	}
}

// トレード計画の検証結果に応じて、キャプテンアメリカのステータスを更新します。
func (c CaptainAmerica) HandleCaptainAmericaStatus(captainAmericaStatus *domain.CaptainAmericaStatus, tradePlan bool) {
	DB := c.DB.Connect()
	params := make(map[string]interface{})
	// セットアップ済みの売買ルールを完了、取引ステータスを取引中に更新する。
	if tradePlan {
		params["setup_status"] = false
		params["trade_status"] = true
		params["second_judge"] = false
	} else {
		if captainAmericaStatus.SecondJudge {
			params["second_judge"] = false
			params["setup_status"] = false
		} else {
			params["second_judge"] = true
		}
	}
	c.CaptainAmericaStatus.Update(DB, captainAmericaStatus, params)
}

// TradeRuleStatusを作成します。
// 既に同一銘柄、足種で作成済みの場合は更新します。
func (c CaptainAmerica) CreateOrUpdateTradeRuleStatus(tradeRuleStatus *domain.TradeRuleStatus) {
	DB := c.DB.Connect()
	if target := c.TradeRuleStatus.FindByTradeRuleAndInstrumentAndGranularity(DB, tradeRuleStatus.TradeRule, tradeRuleStatus.Instrument, tradeRuleStatus.Granularity); target.ID == 0 {
		c.TradeRuleStatus.Create(DB, tradeRuleStatus)
	} else {
		params := map[string]interface{}{
			"candle_time": tradeRuleStatus.CandleTime,
			"status":      tradeRuleStatus.Status,
		}
		c.TradeRuleStatus.Update(DB, &target, params)
	}
}

// TradeRuleStatusを完了として無効にします。
func (c CaptainAmerica) CompleteTradeRuleStatus(tradeRuleStatus *domain.TradeRuleStatus) {
	DB := c.DB.Connect()
	params := map[string]interface{}{
		"status": false,
	}
	c.TradeRuleStatus.Update(DB, tradeRuleStatus, params)
}