package usecase

import (
	"context"
	"fmt"
	"github.com/fmyaaaaaaa/Alice/alice-trading/domain"
	"github.com/fmyaaaaaaa/Alice/alice-trading/domain/enum"
	"github.com/fmyaaaaaaa/Alice/alice-trading/infrastructure/cache"
	"github.com/fmyaaaaaaa/Alice/alice-trading/interfaces/api/msg"
	"github.com/fmyaaaaaaa/Alice/alice-trading/usecase/dto"
	"github.com/fmyaaaaaaa/Alice/alice-trading/usecase/util"
	"log"
)

// 足データのユースケース
type CandlesInteractor struct {
	DB      DBRepository
	Candles CandlesRepository
	Api     CandlesApi
}

// APIで足データを取得し、DBに保存します。
// システム起動時のみ利用します。
func (c CandlesInteractor) InitializeCandle(instrument domain.Instruments, granularity enum.Granularity) (needJudge bool) {
	return c.GetOrLoadCandles(dto.NewCandlesGetDto(instrument.Instrument, 5, granularity))
}

// 足データのキャッシュを構築します。
// DBにデータが存在する場合は読み込み、存在しない場合はAPIから取得します。
func (c CandlesInteractor) GetOrLoadCandles(dto dto.CandlesGetDto) (needJudge bool) {
	candles := c.GetCandles(dto.InstrumentName, dto.Granularity)
	if len(candles) == 0 {
		candles = c.GetCandle(dto)
		needJudge = true
	}
	cacheName := fmt.Sprintf("candles-%s-%s", dto.InstrumentName, dto.Granularity)
	cacheManager := cache.GetCacheManager()
	cacheManager.Set(cacheName, candles, enum.NoExpiration)
	return needJudge
}

// 足データをAPIを実行し取得します。
func (c CandlesInteractor) GetCandle(candlesGetDto dto.CandlesGetDto) []domain.BidAskCandles {
	res, _ := c.Api.GetCandleBidAsk(context.Background(), candlesGetDto.InstrumentName, candlesGetDto.Count, candlesGetDto.Granularity)
	return c.convertToInitialEntity(res, candlesGetDto.InstrumentName, candlesGetDto.Granularity)
}

// 足データをDBから取得します。
func (c CandlesInteractor) GetCandles(instrument string, granularity enum.Granularity) []domain.BidAskCandles {
	DB := c.DB.Connect()
	return c.Candles.FindByInstrumentAndGranularity(DB, instrument, granularity)
}

// 引数の足データを保存します。
func (c CandlesInteractor) Create(candle *domain.BidAskCandles) {
	db := c.DB.Connect()
	c.Candles.Create(db, candle)
}

// 引数の足データを一括保存します。
func (c CandlesInteractor) CreateBulkCandles(candles []domain.BidAskCandles) {
	db := c.DB.Connect()
	c.Candles.BulkCreate(db, &candles)
}

// 主キーをもとに足データを取得します。
func (c CandlesInteractor) Get(id int) (domain.BidAskCandles, error) {
	db := c.DB.Connect()
	result, err := c.Candles.FindByID(db, id)
	if err != nil {
		log.Print("something wrong")
	}
	return result, nil
}

// 足データを全件取得します。
func (c CandlesInteractor) GetAll() (candleList []domain.BidAskCandles) {
	db := c.DB.Connect()
	result := c.Candles.FindAll(db)
	return result
}

// 足データを削除します。
// 主キーが必要です。
func (c CandlesInteractor) Delete(candle *domain.BidAskCandles) {
	db := c.DB.Connect()
	c.Candles.Delete(db, candle)
}

// 初期化時のみの利用を想定しています。また、最後の足データは返却しません。
func (c CandlesInteractor) convertToInitialEntity(res *msg.CandlesBidAskResponse, instrumentName string, granularity enum.Granularity) []domain.BidAskCandles {
	var result []domain.BidAskCandles
	for i, candle := range res.Candles {
		if i == len(res.Candles)-1 {
			continue
		}
		target := &domain.BidAskCandles{
			InstrumentName: instrumentName,
			Granularity:    granularity,
			Bid: domain.BidRate{
				Open:  util.ParseFloat(candle.Bid.O),
				Close: util.ParseFloat(candle.Bid.C),
				High:  util.ParseFloat(candle.Bid.H),
				Low:   util.ParseFloat(candle.Bid.L),
			},
			Ask: domain.AskRate{
				Open:  util.ParseFloat(candle.Ask.O),
				Close: util.ParseFloat(candle.Ask.C),
				High:  util.ParseFloat(candle.Ask.H),
				Low:   util.ParseFloat(candle.Ask.L),
			},
			Candles: domain.Candles{
				Time:   candle.Time,
				Volume: util.ParseFloat(candle.Volume.String()),
			},
		}
		result = append(result, *target)
	}
	return result
}
