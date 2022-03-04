package benchmark

import (
	"fmt"
	"github.com/bitxhub/bitxhub-order-rbft/rbft"
	"github.com/meshplus/bitxhub-core/order"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMulti_Rbft_Node(t *testing.T) {
	peerCnt := 4
	swarms, nodes := newSwarms(t, peerCnt, true)
	defer stopSwarms(t, swarms)

	repoRoot, err := ioutil.TempDir("", "nodes")
	defer os.RemoveAll(repoRoot)

	fileData, err := ioutil.ReadFile("../../../config/order.toml")
	require.Nil(t, err)

	orders := make([]order.Order, 0)
	for i := 0; i < peerCnt; i++ {
		nodePath := fmt.Sprintf("node%d", i)
		nodeRepo := filepath.Join(repoRoot, nodePath)
		err := os.Mkdir(nodeRepo, 0744)
		require.Nil(t, err)
		orderPath := filepath.Join(nodeRepo, "order.toml")
		err = ioutil.WriteFile(orderPath, fileData, 0744)
		require.Nil(t, err)

		ID := i + 1
		rbft, err := rbft.NewNode(
			order.WithRepoRoot(nodeRepo),
			order.WithID(uint64(ID)),
			order.WithNodes(nodes),
			order.WithPeerManager(swarms[i]),
			order.WithStoragePath(repo.GetStoragePath(nodeRepo, "order")),
			order.WithLogger(log.NewWithModule("consensus")),
			order.WithGetBlockByHeightFunc(nil),
			order.WithApplied(1),
			order.WithGetAccountNonceFunc(func(address *types.Address) uint64 {
				return 0
			}),
		)
		require.Nil(t, err)
		err = rbft.Start()
		require.Nil(t, err)
		orders = append(orders, rbft)
		go listen(t, rbft, swarms[i])
	}

	for {
		time.Sleep(2 * time.Second)
		err := orders[0].Ready()
		if err == nil {
			break
		}
	}
	go caldelay(t, orders[0])

	sendTx(t, 1000000, orders[0])
	time.Sleep(20 * time.Second)

}

func sendTx(t *testing.T, count int, node order.Order) {
	privKey, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	for i := 0; i < count; i++ {
		tx := generateTx(privKey, uint64(i))
		err := node.Prepare(tx)
		require.Nil(t, err)
		//fmt.Printf("sendTx%d succuss\n", i)
	}
}

func caldelay(t *testing.T, node order.Order) {
	for {
		select {
		case commitEvent := <-node.Commit():
			var start int64
			var sum int64
			end := commitEvent.Block.BlockHeader.Timestamp
			for _, tx := range commitEvent.Block.Transactions.Transactions {
				start = tx.GetTimeStamp()
				sum += end - start
			}
			delay := sum / int64(len(commitEvent.Block.Transactions.Transactions))
			fmt.Printf("!!!!!!block %d average delay is %d \n", commitEvent.Block.BlockHeader.Number, delay)
			fmt.Printf("tx num is %d \n", len(commitEvent.Block.Transactions.Transactions))

		}
	}
}
