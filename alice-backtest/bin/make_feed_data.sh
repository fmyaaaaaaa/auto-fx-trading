#!/bin/bash
# 実行時にoandaのAPI仕様に基づいて、以下の引数を指定してください。
# 銘柄(instrument) 足種(granularity) 本数(count) AccessToken

# 引数で指定したパラメータ
TARGET_PRODUCT=$1
TARGET_GRANULARITY=$2
COUNT=$3
ACCESS_TOKEN=$4

# 出力先のディレクトリ
OUTPUT_DIR='./../data'

# APIエンドポイント
HEADER_CONTENT_TYPE="Content-Type: application/json"
HEADER_AUTH="Authorization: Bearer $ACCESS_TOKEN"
URL_TO_GET_CANDLES="https://api-fxpractice.oanda.com/v3/instruments/$TARGET_PRODUCT/candles?count=$COUNT&price=M&granularity=$TARGET_GRANULARITY"

# カールを実行し結果をcsv形式で格納
curl -X GET -H "$HEADER_CONTENT_TYPE" -H "$HEADER_AUTH" "$URL_TO_GET_CANDLES" | \
 jq -r '["datetime", "open", "high", "low", "close", "adj close", "volume"], (.candles[]|[.time, .mid["o"], .mid["h"], .mid["l"], .mid["c"], .mid["c"], .volume])|@csv' | \
 sed 's/\"//g' > "$OUTPUT_DIR/candles-$TARGET_PRODUCT-$TARGET_GRANULARITY.csv"

## トレード計画検証用のcsvには以下のコマンドを実行する。（銘柄名、足種は適宜修正が必要。）
# sed -e "/09:00:00.000000000Z/d" candles-USD_JPY-H12.csv | sed "/10:00:00.000000000Z/d" > candles-USD_JPY-H12_eliminate.csv