package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/tendermint/tendermint/libs/bytes"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

// TmQuerier queries tendermint for commitments and events.
type TmQuerier struct {
	logger        tmlog.Logger
	tendermintRPC string
	clientConn    client.Client
}

func NewTmQuerier(
	tendermintRPC string,
	logger tmlog.Logger,
) *TmQuerier {
	return &TmQuerier{
		logger:        logger,
		tendermintRPC: tendermintRPC,
	}
}

func (tq *TmQuerier) Start() error {
	// creating an RPC connection to tendermint
	trpc, err := http.New(tq.tendermintRPC, "/websocket")
	if err != nil {
		return err
	}
	err = trpc.Start()
	if err != nil {
		return err
	}
	tq.clientConn = trpc
	return nil
}

func (tq *TmQuerier) Stop() error {
	err := tq.clientConn.Stop()
	if err != nil {
		return err
	}
	return nil
}

func (tq *TmQuerier) WithClientConn(trpc client.Client) {
	tq.clientConn = trpc
}

func (tq *TmQuerier) QueryCommitment(ctx context.Context, beginBlock uint64, endBlock uint64) (bytes.HexBytes, error) {
	dcResp, err := tq.clientConn.DataCommitment(ctx, beginBlock, endBlock)
	if err != nil {
		return nil, err
	}
	return dcResp.DataCommitment, nil
}

func (tq *TmQuerier) QueryHeight(ctx context.Context) (int64, error) {
	status, err := tq.clientConn.Status(ctx)
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

func (tq *TmQuerier) WaitForHeight(ctx context.Context, height int64) error {
	currentHeight, err := tq.QueryHeight(ctx)
	if err != nil {
		return err
	}
	if currentHeight >= height {
		return nil
	}

	query := fmt.Sprintf("%s='%s'", types.EventTypeKey, types.EventNewBlock)
	results, err := tq.SubscribeEvents(ctx, "sub-height", query)
	if err != nil {
		return err
	}
	defer func() {
		err := tq.UnsubscribeEvents(ctx, "sub-height", query)
		if err != nil {
			tq.logger.Error(err.Error())
		}
	}()

	timeout := time.NewTimer(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return ErrCouldntReachSpecifiedHeight
		case <-results:
			currentHeight, err := tq.QueryHeight(ctx)
			if err != nil {
				return err
			}
			if currentHeight >= height {
				return nil
			}
		}
	}
}

func (tq *TmQuerier) SubscribeEvents(ctx context.Context, subscriptionName string, query string) (<-chan coretypes.ResultEvent, error) {
	// This doesn't seem to complain when the node is down
	results, err := tq.clientConn.Subscribe(
		ctx,
		subscriptionName,
		query,
	)
	if err != nil {
		return nil, err
	}
	return results, err
}

func (tq *TmQuerier) UnsubscribeEvents(ctx context.Context, subscriptionName string, query string) error {
	return tq.clientConn.Unsubscribe(
		ctx,
		subscriptionName,
		query,
	)
}

func (tq *TmQuerier) IsRunning(ctx context.Context) bool {
	_, err := tq.clientConn.Status(ctx)
	return err == nil
}

func (tq *TmQuerier) Reconnect() error {
	_ = tq.clientConn.Stop()
	newConnection, err := http.New(tq.tendermintRPC, "/websocket")
	if err != nil {
		return err
	}
	err = newConnection.Start()
	if err != nil {
		return err
	}
	tq.clientConn = newConnection
	return nil
}
