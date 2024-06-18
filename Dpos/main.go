package main

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "math/rand"
    "time"
)

type Transaction struct {
    From      string
    To        string
    Amount    int
    Signature string
}

type Vote struct {
    Candidate      string
    VoteCount      int
    AvailableVotes int
}

type Node struct {
    Address        string
    AvailableVotes int
    VoteCount      int
    TokenAmount    int
}

type Block struct {
    Index         int
    Timestamp     int64
    Transactions  []Transaction
    Votes         []Vote
    PreviousHash  string
    Hash          string
    Nonce         int
}

type Blockchain struct {
    Nodes        []Node
    Votes        []Vote
    Blocks       []Block
    Transactions []Transaction
}

func calculateHash(block Block) string {
    record := string(block.Index) + string(block.Timestamp) + block.PreviousHash + string(block.Nonce)
    h := sha256.New()
    h.Write([]byte(record))
    hashed := h.Sum(nil)
    return hex.EncodeToString(hashed)
}

func (bc *Blockchain) createGenesisBlock() {
    transactions := []Transaction{}
    votes := []Vote{}
    genesisBlock := Block{
        Index:        0,
        Timestamp:    time.Now().Unix(),
        Transactions: transactions,
        Votes:        votes,
        PreviousHash: "",
        Nonce:        0,
    }
    genesisBlock.Hash = calculateHash(genesisBlock)
    bc.Blocks = append(bc.Blocks, genesisBlock)
    fmt.Println("Genesis Block created:", genesisBlock.Hash)
}

func (bc *Blockchain) addNode(node Node) {
    bc.Nodes = append(bc.Nodes, node)
}

func (bc *Blockchain) addVote(vote Vote) {
    bc.Votes = append(bc.Votes, vote)
}

func initializeNodesAndVotes(bc *Blockchain) {
    rand.Seed(time.Now().UnixNano())
    totalTokens := 10000
    for i := 0; i < 100; i++ {
        tokenAmount := 1 + rand.Intn(totalTokens/10)
        totalTokens -= tokenAmount
        nodeAddress := calculateHash(Block{Index: i})
        node := Node{
            Address:     nodeAddress,
            TokenAmount: tokenAmount,
        }
        bc.addNode(node)
        vote := Vote{
            Candidate:      nodeAddress,
            VoteCount:      tokenAmount,
            AvailableVotes: tokenAmount,
        }
        bc.addVote(vote)
        fmt.Printf("Node %d added: %s, Token Amount: %d\n", i+1, node.Address, node.TokenAmount)
    }
}

func (n *Node) vote(vote *Vote) {
    vote.VoteCount++
    n.AvailableVotes--
}

func simulateVoting(bc *Blockchain) {
    rand.Seed(time.Now().UnixNano())
    for i := range bc.Nodes {
        numVotes := bc.Nodes[i].TokenAmount
        for j := 0; j < numVotes; j++ {
            candidateIndex := rand.Intn(len(bc.Votes))
            bc.Nodes[i].vote(&bc.Votes[candidateIndex])
        }
    }
}

func sortNodesByVoteCount(nodes []Node) []Node {
    sortedNodes := make([]Node, len(nodes))
    copy(sortedNodes, nodes)
    for i := 0; i < len(sortedNodes); i++ {
        for j := i + 1; j < len(sortedNodes); j++ {
            if sortedNodes[i].VoteCount < sortedNodes[j].VoteCount {
                sortedNodes[i], sortedNodes[j] = sortedNodes[j], sortedNodes[i]
            }
        }
    }
    return sortedNodes
}

func printTopNodes(nodes []Node, top int) {
    fmt.Printf("Top %d nodes by votes:\n", top)
    for i := 0; i < top && i < len(nodes); i++ {
        fmt.Printf("%d. %s - Votes: %d\n", i+1, nodes[i].Address, nodes[i].VoteCount)
    }
}

func (bc *Blockchain) addBlock(block Block) {
    block.Hash = calculateHash(block)
    bc.Blocks = append(bc.Blocks, block)
    fmt.Printf("Block %d added with hash: %s\n", block.Index, block.Hash)
}

func (bc *Blockchain) validate() bool {
    for i := 1; i < len(bc.Blocks); i++ {
        if bc.Blocks[i].PreviousHash != bc.Blocks[i-1].Hash {
            return false
        }
    }
    return true
}

func main() {
    blockchain := Blockchain{}
    blockchain.createGenesisBlock()
    initializeNodesAndVotes(&blockchain)
    simulateVoting(&blockchain)
    sortedNodes := sortNodesByVoteCount(blockchain.Nodes)
    printTopNodes(sortedNodes, 30)

    newBlock1 := Block{
        Index:        1,
        Timestamp:    time.Now().Unix(),
        Transactions: []Transaction{},
        Votes:        blockchain.Votes,
        PreviousHash: blockchain.Blocks[len(blockchain.Blocks)-1].Hash,
        Nonce:        0,
    }
    blockchain.addBlock(newBlock1)

    newBlock2 := Block{
        Index:        2,
        Timestamp:    time.Now().Unix(),
        Transactions: []Transaction{},
        Votes:        blockchain.Votes,
        PreviousHash: blockchain.Blocks[len(blockchain.Blocks)-1].Hash,
        Nonce:        0,
    }
    blockchain.addBlock(newBlock2)

    isValid := blockchain.validate()
    fmt.Printf("Blockchain valid: %t\n", isValid)
}
