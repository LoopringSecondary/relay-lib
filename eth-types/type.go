package types

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

//go:generate gencodec -type Order -field-override orderMarshaling -out gen_order_json.go
type Order struct {
	Protocol              common.Address `json:"protocol" gencodec:"required"` // 智能合约地址
	AuthAddr              common.Address `json:"authAddr" gencodec:"required"` //
	WalletId              *big.Int       `json:"walletId" gencodec:"required"`
	TokenS                common.Address `json:"tokenS" gencodec:"required"`     // 卖出erc20代币智能合约地址
	TokenB                common.Address `json:"tokenB" gencodec:"required"`     // 买入erc20代币智能合约地址
	AmountS               *big.Int       `json:"amountS" gencodec:"required"`    // 卖出erc20代币数量上限
	AmountB               *big.Int       `json:"amountB" gencodec:"required"`    // 买入erc20代币数量上限
	ValidSince            *big.Int       `json:"validSince" gencodec:"required"` //
	ValidUntil            *big.Int       `json:"validUntil" gencodec:"required"` // 订单过期时间
	LrcFee                *big.Int       `json:"lrcFee" `                        // 交易总费用,部分成交的费用按该次撮合实际卖出代币额与比例计算
	BuyNoMoreThanAmountB  bool           `json:"buyNoMoreThanAmountB" gencodec:"required"`
	MarginSplitPercentage uint8          `json:"marginSplitPercentage" gencodec:"required"` // 不为0时支付给交易所的分润比例，否则视为100%
	V                     uint8          `json:"v" gencodec:"required"`
	R                     Bytes32        `json:"r" gencodec:"required"`
	S                     Bytes32        `json:"s" gencodec:"required"`
	Price                 *big.Rat       `json:"price"`
	Owner                 common.Address `json:"owner"`
	Hash                  common.Hash    `json:"hash"`
	Market                string         `json:"market"`
}

type TokenRegisterEvent struct {
	Token  common.Address
	Symbol string
}

type TokenUnRegisterEvent struct {
	Token  common.Address
	Symbol string
}

type AddressAuthorizedEvent struct {
	Protocol common.Address
	Number   int
}

type AddressDeAuthorizedEvent struct {
	Protocol common.Address
	Number   int
}

type TransferEvent struct {
	Sender   common.Address
	Receiver common.Address
	Value    *big.Int
}

type ApprovalEvent struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
}

type OrderFilledEvent struct {
	Ringhash      common.Hash
	PreOrderHash  common.Hash
	OrderHash     common.Hash
	NextOrderHash common.Hash
	Owner         common.Address
	TokenS        common.Address
	TokenB        common.Address
	SellTo        common.Address
	BuyFrom       common.Address
	RingIndex     *big.Int
	AmountS       *big.Int
	AmountB       *big.Int
	LrcReward     *big.Int
	LrcFee        *big.Int
	SplitS        *big.Int
	SplitB        *big.Int
	Market        string
	FillIndex     *big.Int
}

type OrderCancelledEvent struct {
	OrderHash       common.Hash
	AmountCancelled *big.Int
}

type CutoffEvent struct {
	Owner         common.Address
	Cutoff        *big.Int
	OrderHashList []common.Hash
}

type CutoffPairEvent struct {
	Owner         common.Address
	Token1        common.Address
	Token2        common.Address
	Cutoff        *big.Int
	OrderHashList []common.Hash
}

type RingMinedEvent struct {
	RingIndex    *big.Int
	TotalLrcFee  *big.Int
	TradeAmount  int
	Ringhash     common.Hash
	Miner        common.Address
	FeeRecipient common.Address
}

type WethDepositEvent struct {
	Dst   common.Address
	Value *big.Int
}

type WethWithdrawalEvent struct {
	Src   common.Address
	Value *big.Int
}

type WethDepositMethodEvent struct {
	Dst   common.Address
	Value *big.Int
}

type WethWithdrawalMethodEvent struct {
	Src   common.Address
	Value *big.Int
}

type ApproveMethodEvent struct {
	Spender common.Address
	Value   *big.Int
	Owner   common.Address
}

type TransferMethodEvent struct {
	Sender   common.Address
	Receiver common.Address
	Value    *big.Int
}

type CutoffMethodEvent struct {
	Value *big.Int
	Owner common.Address
}

type CutoffPairMethodEvent struct {
	Value  *big.Int
	Token1 common.Address
	Token2 common.Address
	Owner  common.Address
}
