package coreapi

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/model/events"
)

type FeedAPI CoreAPI

var _ api.FeedAPI = (*FeedAPI)(nil)

func (api *FeedAPI) SubscribeNewBlockEvent(ch chan<- events.ExecutedEvent) event.Subscription {
	return api.bxh.BlockExecutor.SubscribeBlockEvent(ch)
}

func (api *FeedAPI) SubscribeReceiptEvent(txHash string, ch chan<- *pb.Receipt) event.Subscription {
	return api.bxh.BlockExecutor.SubscribeReceiptEvent(txHash, ch)
}
