package rule

import (
	"github.com/fmyaaaaaaa/Alice/alice-trading/domain"
	"github.com/fmyaaaaaaa/Alice/alice-trading/domain/enum"
	"github.com/fmyaaaaaaa/Alice/alice-trading/infrastructure/config"
	"github.com/fmyaaaaaaa/Alice/alice-trading/usecase"
	"log"
	"strconv"
)

type IronMan struct {
	DB                usecase.DBRepository
	SwingHighLowPrice usecase.SwingHighLowPriceRepository
	SwingTarget       usecase.SwingTargetRepository
	IronManStatus     usecase.IronManStatusRepository
	TradeRuleStatus   usecase.TradeRuleStatusRepository
}

// 足データをもとにセットアップを検証します。
func (i IronMan) JudgementSetup(currentCandle *domain.BidAskCandles, instrument string, granularity enum.Granularity) {
	// セットアップの検証対象となる高値、安値を取得する。
	swingTarget := i.GetSwingTargetForSetUp(instrument, granularity)
	targetHighLowPrice := i.GetHighLowPrice(swingTarget.SwingID)

	setUp := false
	var ironManStatus *domain.IronManStatus
	switch {
	// 上昇トレンドかつ、仲値がターゲットの高値を超えた場合 または 下降トレンドかつ、仲値がターゲットの安値を超えた場合
	case currentCandle.Trend == enum.UpTrend && targetHighLowPrice.HighPrice <= currentCandle.GetAveMid(),
		currentCandle.Trend == enum.DownTrend && targetHighLowPrice.LowPrice >= currentCandle.GetAveMid():
		setUp = true
	}

	// セットアップ情報を更新する。同一スイングでセットアップを作成済みの場合はスキップする。
	if setUp {
		ironManStatus = domain.NewIronManStatus(instrument, granularity, swingTarget.ID, currentCandle.Trend)
		if ok := i.CreateIronManStatus(ironManStatus); ok {
			tradeRuleStatus := domain.NewTradeRuleStatus(enum.IronMan, instrument, granularity, currentCandle.Candles.Time)
			i.CreateTradeRuleStatus(tradeRuleStatus)
			log.Println("IronMan setup happened ", instrument, granularity)
		}
	}
}

// 足データをもとにトレード計画を判定します。
func (i IronMan) JudgementTradePlan(tradeRuleStatus domain.TradeRuleStatus, candle *domain.BidAskCandles, instrument string, granularity enum.Granularity) (bool, string) {
	// 注文数量
	units := 0
	// セットアップと同一の足データの場合は処理をスキップ。
	// トレード計画の判定はセットアップの次回足データを対象とするため。
	if tradeRuleStatus.CandleTime.Equal(candle.Candles.Time) {
		return false, strconv.Itoa(units)
	}
	// セットアップ情報を取得する。
	ironManStatus := i.GetIronManStatus(instrument, granularity)
	swingTarget := i.GetSwingTargetForTradePlan(ironManStatus.SwingTargetID)
	highLowPrice := i.GetHighLowPrice(swingTarget.SwingID)
	tradePlan := false
	// TODO:資金管理から数量、トレーリングストップ値幅を取得し、OrderManagerから注文を実行する
	switch {
	case ironManStatus.Trend == enum.UpTrend && highLowPrice.HighPrice <= candle.GetAveMid():
		tradePlan = true
		units = config.GetInstance().Property.OrderLot
		log.Println("IronMan trade happened", instrument, granularity, candle.GetAveMid())
	case ironManStatus.Trend == enum.DownTrend && highLowPrice.LowPrice >= candle.GetAveMid():
		tradePlan = true
		units = -config.GetInstance().Property.OrderLot
		log.Println("IronMan trade happened", instrument, granularity, candle.GetAveMid())
	}
	// セットアップ済みの売買ルールを完了状態にする。
	if tradePlan {
		i.CompleteIronManStatus(&ironManStatus)
		i.CompleteTradeRuleStatus(&tradeRuleStatus)
	}
	return tradePlan, strconv.Itoa(units)
}

// SwingHighLowPriceを取得します。
func (i IronMan) GetHighLowPrice(swingID int) domain.SwingHighLowPrice {
	DB := i.DB.Connect()
	return i.SwingHighLowPrice.FindBySwingID(DB, swingID)
}

// SwingTargetを取得します。(セットアップ検証）
func (i IronMan) GetSwingTargetForSetUp(instrument string, granularity enum.Granularity) domain.SwingTarget {
	DB := i.DB.Connect()
	return i.SwingTarget.FindByInstrumentAndGranularity(DB, instrument, granularity)
}

// SwingTargetを取得します。（トレード計画検証）
func (i IronMan) GetSwingTargetForTradePlan(id int) domain.SwingTarget {
	DB := i.DB.Connect()
	return i.SwingTarget.FindByID(DB, id)
}

// IronManStatusを取得します。
func (i IronMan) GetIronManStatus(instrument string, granularity enum.Granularity) domain.IronManStatus {
	DB := i.DB.Connect()
	return i.IronManStatus.FindByInstrumentAndGranularity(DB, instrument, granularity)
}

// IronManStatusを完了にします。
func (i IronMan) CompleteIronManStatus(ironManStatus *domain.IronManStatus) {
	DB := i.DB.Connect()
	params := map[string]interface{}{
		"status": false,
	}
	i.IronManStatus.Update(DB, ironManStatus, params)
}

// IronManStatusを作成します。
// 同じSwingTargetIDかつ、同一トレンドで既にセットアップ済みの場合はスキップします。
func (i IronMan) CreateIronManStatus(ironManStatus *domain.IronManStatus) bool {
	DB := i.DB.Connect()
	check := i.IronManStatus.FindByInstrumentAndGranularity(DB, ironManStatus.Instrument, ironManStatus.Granularity)
	if check.SwingTargetID == ironManStatus.SwingTargetID && check.Trend == ironManStatus.Trend {
		return false
	}
	i.IronManStatus.Create(DB, ironManStatus)
	return true
}

// TradeRuleStatusを作成します。
func (i IronMan) CreateTradeRuleStatus(tradeRuleStatus *domain.TradeRuleStatus) {
	DB := i.DB.Connect()
	i.TradeRuleStatus.Create(DB, tradeRuleStatus)
}

// TradeRuleStatusを完了として無効にします。
func (i IronMan) CompleteTradeRuleStatus(tradeRuleStatus *domain.TradeRuleStatus) {
	DB := i.DB.Connect()
	params := map[string]interface{}{
		"status": false,
	}
	i.TradeRuleStatus.Update(DB, tradeRuleStatus, params)
}
