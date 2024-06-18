package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// POW算法实验指导书中的数据结构和函数实现

var (
	maxNonce = math.MaxInt64
)

// Block 自定义区块结构
type Block struct {
	*BlockWithoutProof
	Proof
}

// Proof 区块的证明信息
type Proof struct {
	ActualTimestamp int64  `json:"actualTimestamp"`
	Nonce           int64  `json:"nonce"`
	hash            []byte
	HashHex         string `json:"hashHex"`
}

// BlockWithoutProof 不带证明信息的区块
type BlockWithoutProof struct {
	CoinBase           int64   `json:"coinBase"`
	timestamp          int64
	data               []byte
	prevBlockHash      []byte
	PrevBlockHashHex   string  `json:"prevBlockHashHex"`
	TargetBit          float64 `json:"targetBit"`
}

// Miner 矿工结构
type Miner struct {
	Id            int64          `json:"id"`
	Balance       uint           `json:"balance"`
	blockchain    *Blockchain
	waitForSignal chan interface{} `json:"-"`
}

// Blockchain 区块链数据
type Blockchain struct {
	config           BlockchainConfig
	currentDifficulty float64
	blocks            []Block
	miners            []Miner
	mutex             *sync.RWMutex
}

// BlockchainConfig 区块链配置信息
type BlockchainConfig struct {
	MinerCount                 int
	OutBlockTime               uint
	InitialDifficulty          float64
	ModifyDifficultyBlockNumber uint
	BookkeepingIncentives      uint
}

// BlockchainInfo 区块链信息
type BlockchainInfo struct {
	Blocks []*Block `json:"blocks"`
	Miners []*Miner `json:"miners"`
}

func main() {
	var count int
	fmt.Printf("请输入初始矿工数量：")
	fmt.Scanf("%d", &count)
	time.Sleep(10 * time.Second)
	fmt.Printf("开始挖矿\n")
	work := NewBlockChainNetWork(BlockchainConfig{
		MinerCount:                 count,
		OutBlockTime:               10,
		InitialDifficulty:          20,
		ModifyDifficultyBlockNumber: 10,
		BookkeepingIncentives:      20,
	})
	work.RunBlockChainNetWork()
	RunRouter(work)
}

// NewBlockChainNetWork 新建一个区块链网络
func NewBlockChainNetWork(blockchainConfig BlockchainConfig) *Blockchain {
	b := &Blockchain{
		config:           blockchainConfig,
		mutex:            &sync.RWMutex{},
		currentDifficulty: blockchainConfig.InitialDifficulty,
	}
	b.blocks = append(b.blocks, *GenerateGenesisBlock([]byte("")))
	for i := 0; i < blockchainConfig.MinerCount; i++ {
		miner := Miner{
			Id:            int64(i),
			Balance:       0,
			blockchain:    b,
			waitForSignal: make(chan interface{}, 1),
		}
		b.miners = append(b.miners, miner)
	}
	return b
}

// GenerateGenesisBlock 生成创世区块
func GenerateGenesisBlock(data []byte) *Block {
	b := &Block{BlockWithoutProof: &BlockWithoutProof{}}
	b.ActualTimestamp = time.Now().Unix()
	b.data = data
	return b
}

// RunBlockChainNetWork 运行区块链网络
func (b *Blockchain) RunBlockChainNetWork() {
	for _, m := range b.miners {
		go m.run()
	}
}

// run 矿工挖矿逻辑
func (m Miner) run() {
	count := 0
	for {
		// 生成
		blockWithoutProof := m.blockchain.assembleNewBlock(m.Id, []byte(fmt.Sprintf("模拟区块数据:%d:%d", m.Id, count)))
		block, finish := blockWithoutProof.Mine(m.waitForSignal)
		if !finish {
			continue
		} else {
			m.blockchain.AddBlock(block, m.waitForSignal)
		}
		count++
	}
}

// assembleNewBlock 组装新的区块
func (b *Blockchain) assembleNewBlock(coinBase int64, data []byte) BlockWithoutProof {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	proof := BlockWithoutProof{
		CoinBase:         coinBase,
		timestamp:        time.Now().Unix(),
		data:             data,
		prevBlockHash:    b.blocks[len(b.blocks)-1].hash,
		TargetBit:        b.currentDifficulty,
		PrevBlockHashHex: b.blocks[len(b.blocks)-1].HashHex,
	}
	return proof
}

// Mine 挖矿函数
func (b *BlockWithoutProof) Mine(waitForSignal chan interface{}) (*Block, bool) {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-b.TargetBit))

	var hashInt big.Int
	var hash [32]byte
	nonce := 0
	for nonce != maxNonce {
		select {
		case <-waitForSignal:
			return nil, false
		default:
			data := b.prepareData(int64(nonce))
			hash = sha256.Sum256(data)
			hashInt.SetBytes(hash[:])
			if hashInt.Cmp(target) < 0 {
				block := &Block{
					BlockWithoutProof: b,
					Proof: Proof{
						Nonce:   int64(nonce),
						hash:    hash[:],
						HashHex: hex.EncodeToString(hash[:]),
					},
				}
				return block, true
			} else {
				nonce++
			}
		}
	}
	return nil, false
}

// prepareData 准备数据
func (block *BlockWithoutProof) prepareData(nonce int64) []byte {
	data := bytes.Join(
		[][]byte{
			int2Hex(block.CoinBase),
			block.prevBlockHash,
			block.data,
			int2Hex(block.timestamp),
			int2Hex(int64(block.TargetBit)),
			int2Hex(nonce),
		},
		[]byte{},
	)
	return data
}

// AddBlock 增加一个区块到区块链
func (bc *Blockchain) AddBlock(block *Block, signal chan interface{}) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	block.ActualTimestamp = time.Now().Unix()
	if !bc.verifyNewBlock(block) {
		return
	}

	bc.blocks = append(bc.blocks, *block)
	bc.adjustDifficulty()
	bc.bookkeepingRewards(block.CoinBase)
	bc.notifyMiners(block.CoinBase)
	fmt.Printf(" %s: %d 节点挖出了一个新的区块 %s\n", time.Now(), block.CoinBase, block.HashHex)
}

// verifyNewBlock 验证新区块
func (bc *Blockchain) verifyNewBlock(block *Block) bool {
	prevBlock := bc.blocks[len(bc.blocks)-1]
	if uint64(block.TargetBit) != uint64(bc.currentDifficulty) {
		return false
	}
	if string(prevBlock.hash) != string(block.prevBlockHash) {
		return false
	}
	if !block.Verify() {
		return false
	}
	return true
}



// adjustDifficulty 根据挖矿的时间调整难度值
func (bc *Blockchain) adjustDifficulty() {
	if uint(len(bc.blocks))%bc.config.ModifyDifficultyBlockNumber == 0 {
		block := bc.blocks[len(bc.blocks)-1]
		preDiff := bc.currentDifficulty
		actuallyTime := float64(block.ActualTimestamp - bc.blocks[uint(len(bc.blocks))-bc.config.ModifyDifficultyBlockNumber].ActualTimestamp)
		theoryTime := float64(bc.config.OutBlockTime * bc.config.ModifyDifficultyBlockNumber)
		ratio := theoryTime / actuallyTime
		if ratio > 1.1 {
			ratio = 1.1
		} else if ratio < 0.5 {
			ratio = 0.5
		}
		bc.currentDifficulty = bc.currentDifficulty * ratio
		fmt.Println("难度阈值改变 preDiff:", preDiff, "nowDiff", bc.currentDifficulty)
	}
}

// bookkeepingRewards 给予挖矿成功的矿工奖励
func (bc *Blockchain) bookkeepingRewards(coinBase int64) {
	bc.miners[coinBase].Balance += bc.config.BookkeepingIncentives
}

// notifyMiners 通知所有矿工挖矿成功
func (bc *Blockchain) notifyMiners(sponsor int64) {
	for i, miner := range bc.miners {
		if i != int(sponsor) {
			go func(signal chan interface{}) {
				signal <- struct{}{}
			}(miner.waitForSignal)
		}
	}
}

// RunRouter 运行web服务
func RunRouter(blockchain *Blockchain) {
	r := gin.Default()
	r.GET("/addMiner", addMiner(blockchain))
	r.GET("/getBlockChainInfo", getBlockChainInfo(blockchain))
	r.Run()
}

// addMiner 增加矿工
func addMiner(blockchain *Blockchain) gin.HandlerFunc {
	return func(c *gin.Context) {
		blockchain.IncreaseMiner()
		c.JSON(200, gin.H{
			"message": "增加成功",
		})
	}
}

// getBlockChainInfo 获取区块链信息
func getBlockChainInfo(blockchain *Blockchain) gin.HandlerFunc {
	return func(c *gin.Context) {
		blocks, miners := blockchain.GetBlockInfo()
		c.JSON(200, gin.H{
			"blocks": blocks,
			"miners": miners,
		})
	}
}

// IncreaseMiner 增加矿工
func (bc *Blockchain) IncreaseMiner() bool {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	var miner = Miner{
		Id:            int64(len(bc.miners)),
		Balance:       0,
		blockchain:    bc,
		waitForSignal: make(chan interface{}, 1),
	}
	bc.miners = append(bc.miners, miner)
	go miner.run()
	return true
}

// GetBlockInfo 获取区块信息
func (bc *Blockchain) GetBlockInfo() ([]Block, []Miner) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	blocks := make([]Block, len(bc.blocks))
	miners := make([]Miner, len(bc.miners))
	copy(blocks, bc.blocks)
	copy(miners, bc.miners)
	return blocks, miners
}

// int2Hex 整数转十六进制
func int2Hex(n int64) []byte {
	return []byte(fmt.Sprintf("%x", n))
}
