# GoNES - NES Emulator in Go

Go言語で実装されたファミコン（NES）エミュレータです。

## 対応状況

**注意**: 現在Mapper 0 (NROM)のみ対応しています。Mapper 1以降のゲームは動作しません。

## ビルド方法

```bash
git clone https://github.com/RNG999/gones.git
cd gones
go mod tidy
go build -o gones ./cmd/gones
```

## 使用方法

```bash
# 基本実行
./gones -rom game.nes

# ヘッドレスモード
./gones -rom game.nes -nogui

# デバッグモード
./gones -rom game.nes -debug
```

## 操作方法

| キー | 機能 |
|------|------|
| W/A/S/D | 十字キー（上/左/下/右） |
| J | Aボタン |
| K | Bボタン |
| Enter | Start |
| Shift | Select |

