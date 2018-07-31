package core

import (
	"container/list"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pelletier/go-toml"

	"github.com/spf13/cobra"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/util"
)

type config struct {
	SendAcct      string   `toml:"send_acct_id"`
	Sendasset     string   `toml:"send_asset_id"`
	AssetAddr     string   `toml:"asset_address"`
	BuildType     string   `toml:"build_type"`
	AssetReceiver []string `toml:"asset_receiver"`
	Password      string   `toml:"password"`
}

func init() {
	sendTxCmd.PersistentFlags().IntVar(&thdTxNum, "thdtxnum", 1, " The number of transactions per goroutine")
	sendTxCmd.PersistentFlags().IntVar(&thdNum, "thdnum", 1, "goroutine num")
	sendTxCmd.PersistentFlags().IntVar(&assetNum, "assetnum", 100000000, "Number of transactions asset,unit: neu")
	sendTxCmd.PersistentFlags().IntVar(&mulOutput, "muloutput", 0, "Multiple outputs")
	sendTxCmd.PersistentFlags().StringVar(&configFile, "config", "./config.toml", "config file")

}

var (
	acctNum    int
	thdTxNum   int
	thdNum     int
	assetNum   int
	sendAcct   string
	sendasset  string
	configFile string
	mulOutput  int
	cfg        config
	m          sync.Mutex
	success     = 0
	fail       = 0
)

var sendTxCmd = &cobra.Command{
	Use:   "sendbulktx",
	Short: "send bulk tx",
	Args:  cobra.RangeArgs(0, 4),
	Run: func(cmd *cobra.Command, args []string) {
		bs, err := ioutil.ReadFile(configFile)
		if err = toml.Unmarshal(bs, &cfg); err != nil {
			fmt.Println(err)
			return
		}
		sendAcct = cfg.SendAcct
		sendasset = cfg.Sendasset
		acctNum = len(cfg.AssetReceiver)
		controlPrograms := make([]string, acctNum)
		txidChan := make(chan string)
		switch cfg.BuildType {
		case "issue", "spend", "address":
			for i, value := range cfg.AssetReceiver {
				controlPrograms[i] = value
			}
		default:
			fmt.Println("Invalid transaction template type")
			os.Exit(util.ErrLocalExe)
		}
		txBtm := fmt.Sprintf("%d", assetNum)
		fmt.Println("*****************send tx start*****************")
		// send btm to account
		index := uint64(0)
		for i := 0; i < thdNum; i++ {
			go Sendbulktx(thdTxNum, txBtm, sendAcct, sendasset, controlPrograms, txidChan, &index)
		}

		txs := list.New()
		go recvTxID(txs, txidChan)
		num := 0
		start := time.Now()
		blockTxNum := make(map[uint64]uint32)
		for {
			var n *list.Element
			for e := txs.Front(); e != nil; e = n {
				//fmt.Println(e.Value) //输出list的值,01234
				value := fmt.Sprintf("%s", e.Value)
				param := []string{value}
				if resp, ok := SendReq(GetTransaction, param); ok {
					var tx query.AnnotatedTx
					RestoreStruct(resp, &tx)
					if _, ok := blockTxNum[tx.BlockHeight]; ok {
						blockTxNum[tx.BlockHeight]++
					} else {
						blockTxNum[tx.BlockHeight] = 1
					}
					n = e.Next()
					m.Lock()
					txs.Remove(e)
					m.Unlock()
					num++
					continue
				} else {
					n = e.Next()
				}

			}
			if num >= success && (success+fail) >= thdTxNum*thdNum {
				end := time.Now()
				fmt.Printf("tx num: %d, use time: %v\n", num, end.Sub(start))
				var keys []uint64
				for k := range blockTxNum {
					keys = append(keys, k)
				}
				for _, key := range keys {
					fmt.Println("height:", key, ",tx num:", blockTxNum[key])
				}
				os.Exit(0)
			}
			time.Sleep(time.Second * 60)
		}

	},
}

func recvTxID(txs *list.List, txidChan chan string) {

	file, error := os.OpenFile("./txid.txt", os.O_RDWR|os.O_CREATE, 0766)
	if error != nil {
		fmt.Println(error)
	}

	for {
		select {
		case txid := <-txidChan:
			if strings.EqualFold(txid, "") {
				fail++
			} else {
				success++
				m.Lock()
				txs.PushBack(txid)
				m.Unlock()
				file.WriteString(txid)
				file.WriteString("\n")
			}
		default:
			if fail >= (thdTxNum * thdNum) {
				os.Exit(1)
			}
			if (success + fail) >= (thdTxNum * thdNum) {
				file.Close()
				return
			}

			time.Sleep(time.Second * 1)
		}
	}
}

// Execute send tx
func Execute() {
	if _, err := sendTxCmd.ExecuteC(); err != nil {
		os.Exit(util.ErrLocalExe)
	}
}
