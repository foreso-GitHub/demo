package node

import (
	"errors"
	"fmt"
	"time"

	"github.com/tokentransfer/chain/account"
	"github.com/tokentransfer/chain/block"
	"github.com/tokentransfer/chain/core"

	libblock "github.com/tokentransfer/interfaces/block"
	libcore "github.com/tokentransfer/interfaces/core"
	libcrypto "github.com/tokentransfer/interfaces/crypto"
	libnode "github.com/tokentransfer/interfaces/node"
)

type ConsensusService struct {
	CryptoService libcrypto.CryptoService
	MerkleService libnode.MerkleService
	Config        libcore.Config

	ValidatedBlock libblock.Block
	CurrentBlock   libblock.Block
}

func (service *ConsensusService) GetBlockNumber() int64 {
	if service.ValidatedBlock != nil {
		return int64(service.ValidatedBlock.GetIndex())
	}
	return -1
}

func (service *ConsensusService) GenerateBlock(list []libblock.TransactionWithData) (libblock.Block, error) {
	cs := service.CryptoService

	var b *block.Block
	if service.ValidatedBlock == nil { //genesis
		if len(list) > 0 {
			return nil, errors.New("error genesis block")
		}

		rootKey, err := account.GenerateFamilySeed("masterpassphrase")
		if err != nil {
			return nil, err
		}
		rootAccount, err := rootKey.GetAddress()
		if err != nil {
			return nil, err
		}
		states := []libblock.State{
			&block.CurrencyState{
				State: block.State{
					BlockIndex: uint64(0),
				},
				Account:     rootAccount,
				Sequence:    uint64(0),
				Name:        "TEST Coin",
				Symbol:      "TEST",
				Decimals:    6,
				TotalSupply: int64(110000000000000),
			},
			&block.CurrencyState{
				State: block.State{
					BlockIndex: uint64(0),
				},
				Account:     rootAccount,
				Sequence:    uint64(1),
				Name:        "New Coin",
				Symbol:      "NewCoin",
				Decimals:    8,
				TotalSupply: int64(990000000000000),
			},
			&block.AccountState{
				State: block.State{
					BlockIndex: uint64(0),
				},
				Account:  rootAccount,
				Sequence: uint64(0),
				Amount:   int64(110000000000000),
			},
			&block.DeviceState{
				State: block.State{
					BlockIndex: uint64(0),
				},
				Account:     rootAccount,
				Sequence:    uint64(0),
				Symbol:      "Device_1",
				Description: "Device 1 for station_ppv",
				Tags:        []string{"31.140049", "121.328787", "PVS000001", "huayifristworkshop"},
			},
		}

		ms := service.MerkleService
		for i := 0; i < len(states); i++ {
			state := states[i]
			err := ms.PutState(state)
			if err != nil {
				return nil, err
			}
		}

		b = &block.Block{
			BlockIndex: uint64(0),
			ParentHash: libcrypto.ZeroHash(cs),

			Transactions:    []libblock.TransactionWithData{},
			TransactionHash: ms.GetTransactionRoot(),

			States:    states,
			StateHash: ms.GetStateRoot(),

			Timestamp: time.Now().UnixNano(),
		}

		err = ms.Cancel()
		if err != nil {
			return nil, err
		}
	} else {
		ms := service.MerkleService
		v := service.ValidatedBlock

		fmt.Printf("=== package %d transactions in block %d\n", len(list), v.GetIndex()+1)

		stateMap := map[string][]uint64{}
		for i := 0; i < len(list); i++ {
			txWithData := list[i]

			r := txWithData.GetReceipt()
			r.SetTransactionIndex(uint32(i))
			states := r.GetStates()
			for j := 0; j < len(states); j++ {
				s := states[j]
				s.SetBlockIndex(v.GetIndex() + 1)

				key := fmt.Sprintf("%d-%s", s.GetStateType(), s.GetStateKey())
				index := s.GetIndex()
				stateMap[key] = []uint64{uint64(i), index}
			}

			err := ms.PutTransaction(txWithData)
			if err != nil {
				return nil, err
			}

			fmt.Printf("=== %d %s\n", i, txWithData.GetTransaction().GetHash().String())
		}

		states := make([]libblock.State, 0)
		for i := 0; i < len(list); i++ {
			txWithData := list[i]

			r := txWithData.GetReceipt()
			rs := r.GetStates()
			for j := 0; j < len(rs); j++ {
				s := rs[j]

				key := fmt.Sprintf("%d-%s", s.GetStateType(), s.GetStateKey())
				item, ok := stateMap[key]
				if ok && item[0] == uint64(i) && item[1] == s.GetIndex() {
					states = append(states, s)
				}
			}
		}

		for i := 0; i < len(states); i++ {
			state := states[i]
			err := service.MerkleService.PutState(state)
			if err != nil {
				return nil, err
			}
		}

		b = &block.Block{
			BlockIndex: v.GetIndex() + 1,
			ParentHash: v.GetHash(),

			Transactions:    list,
			TransactionHash: ms.GetTransactionRoot(),

			States:    states,
			StateHash: ms.GetStateRoot(),

			Timestamp: time.Now().UnixNano(),
		}

		err := ms.Cancel()
		if err != nil {
			return nil, err
		}
	}

	_, _, err := cs.Raw(b, libcrypto.RawBinary)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (service *ConsensusService) VerifyBlock(b libblock.Block) (ok bool, err error) {
	ms := service.MerkleService
	cs := service.CryptoService

	ok = true
	err = nil

	defer func() {
		if !ok || err != nil {
			ms.Cancel()
		}
	}()

	transactions := b.GetTransactions()
	l := len(transactions)
	for i := 0; i < l; i++ {
		txWithData := transactions[i]
		tx := txWithData.GetTransaction()

		ok, err = service.VerifyTransaction(tx)
		if err != nil {
			return
		}
		if !ok {
			err = errors.New("verify transaction failed")
			return
		}

		newWithData, e := service.ProcessTransaction(tx)
		if e != nil {
			ok = false
			err = e
			return
		}

		arh, _, e := cs.Raw(txWithData.GetReceipt(), libcrypto.RawBinary)
		if e != nil {
			ok = false
			err = e
			return
		}
		brh, _, e := cs.Raw(newWithData.GetReceipt(), libcrypto.RawBinary)
		if e != nil {
			ok = false
			err = e
			return
		}
		if !arh.Equals(brh) {
			ok = false
			err = errors.New("process transaction receipt failed")
			return
		}

		err = ms.PutTransaction(txWithData)
		if err != nil {
			ok = false
			return
		}
	}

	if service.ValidatedBlock != nil {
		if b.GetIndex() != (service.ValidatedBlock.GetIndex() + 1) {
			ok = false
			err = fmt.Errorf("error block index: %d != %d", b.GetIndex(), (service.ValidatedBlock.GetIndex() + 1))
			return
		}
		if !b.GetParentHash().Equals(service.ValidatedBlock.GetHash()) {
			ok = false
			err = fmt.Errorf("error parent hash: %s != %s", b.GetParentHash().String(), service.ValidatedBlock.GetHash().String())
			return
		}
	} else {
		if b.GetIndex() != 0 {
			ok = false
			err = errors.New("error block index")
			return
		}
		if !b.GetParentHash().IsZero() {
			ok = false
			err = errors.New("error parent hash")
			return
		}
	}

	states := b.GetStates()
	l = len(states)
	for i := 0; i < l; i++ {
		state := states[i]
		err = ms.PutState(state)
		if err != nil {
			ok = false
			return
		}
	}

	transactionHash := ms.GetTransactionRoot()
	stateHash := ms.GetStateRoot()
	if !b.GetTransactionHash().Equals(transactionHash) {
		ok = false
		err = fmt.Errorf("error transaction hash: %s != %s", b.GetTransactionHash().String(), transactionHash.String())
		return
	}
	if !b.GetStateHash().Equals(stateHash) {
		ok = false
		err = fmt.Errorf("error state hash: %s != %s", b.GetStateHash().String(), stateHash.String())
		return
	}
	return
}

func (service *ConsensusService) AddBlock(b libblock.Block) error {
	ms := service.MerkleService

	err := ms.PutBlock(b)
	if err != nil {
		return err
	}
	err = ms.Commit()
	if err != nil {
		return err
	}
	service.ValidatedBlock = b

	return nil
}

func (service *ConsensusService) GetAccount(address string) (*block.AccountState, error) {
	ms := service.MerkleService

	state, err := ms.GetStateByKey(address)
	if err != nil {
		return nil, err
	}
	info, ok := state.(*block.AccountState)
	if !ok {
		return nil, errors.New("error account state")
	}
	return info, nil
}

func (service *ConsensusService) GetCurrency(indexKey string) (*block.CurrencyState, error) {
	ms := service.MerkleService

	state, err := ms.GetStateByKey(indexKey)
	if err != nil {
		return nil, err
	}
	info, ok := state.(*block.CurrencyState)
	if !ok {
		return nil, errors.New("error account state")
	}
	return info, nil
}

func (service *ConsensusService) GetDevice(indexKey string) (*block.DeviceState, error) {
	ms := service.MerkleService

	state, err := ms.GetStateByKey(indexKey)
	if err != nil {
		return nil, err
	}
	info, ok := state.(*block.DeviceState)
	if !ok {
		return nil, errors.New("error account state")
	}
	return info, nil
}

func (service *ConsensusService) VerifyTransaction(t libblock.Transaction) (bool, error) {
	cs := service.CryptoService
	ok, err := cs.Verify(t)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, errors.New("error transaction")
	}
	tx, ok := t.(*block.Transaction)
	if !ok {
		return false, errors.New("error transaction")
	}

	account := tx.Account
	address, err := account.GetAddress()
	if err != nil {
		return false, err
	}
	info, _ := service.GetAccount(address)

	sequence := uint64(1)
	amount := int64(0)
	if info != nil {
		sequence = info.Sequence + 1
		amount = info.Amount
	}

	if tx.Sequence != sequence {
		return false, fmt.Errorf("error sequence: %d != %d", tx.Sequence, sequence)
	}

	if (amount - tx.Amount - int64(tx.Gas)) < 0 {
		return false, errors.New("insuffient amount")
	}

	return true, nil
}

func (service *ConsensusService) addBalance(account libcore.Address, amount int64, isFromAccount bool, sequence uint64) (libblock.State, error) {
	address, err := account.GetAddress()
	if err != nil {
		return nil, err
	}
	info, _ := service.GetAccount(address)
	if info != nil {
		s, err := block.CloneState(info)
		if err != nil {
			return nil, err
		}
		info = s.(*block.AccountState)
		info.Amount = info.Amount + amount
		info.Sequence = sequence
	} else {
		info = &block.AccountState{
			State: block.State{
				StateType: libblock.StateType(core.CORE_ACCOUNT_STATE),
			},

			Account:  account,
			Sequence: uint64(0),
			Amount:   amount,
		}
	}
	return info, nil
}

func (service *ConsensusService) addDevice(account libcore.Address, symbol string, description string, tags []string) (libblock.State, error) {
	device, _ := service.GetDevice(symbol)
	//if err != nil {
	//	return nil, err
	//}
	if device != nil {
		return nil, errors.New("Device [" + symbol + "] has existed!")
	} else {
		device = &block.DeviceState{
			State: block.State{
				BlockIndex: uint64(0),
			},
			Account:     account,
			Sequence:    uint64(0),
			Symbol:      symbol,
			Description: description,
			Tags:        tags,
		}
	}
	return device, nil
}

func (service *ConsensusService) ProcessTransaction(t libblock.Transaction) (libblock.TransactionWithData, error) {
	tx, ok := t.(*block.Transaction)
	if !ok {
		return nil, errors.New("error transaction")
	}

	gasAccount := service.Config.GetGasAccount()
	e1, err := service.addBalance(gasAccount, int64(tx.Gas), false, 0)
	if err != nil {
		return nil, err
	}

	account := tx.Account
	e2, err := service.addBalance(account, -(tx.Amount + int64(tx.Gas)), true, t.GetIndex())
	if err != nil {
		return nil, err
	}

	destination := tx.Destination
	e3, err := service.addBalance(destination, tx.Amount, false, 0)
	if err != nil {
		return nil, err
	}

	r := &block.Receipt{
		TransactionResult: 0,
		States: []libblock.State{
			e1,
			e2,
			e3,
		},
	}

	tx_type := tx.Type
	if tx_type == "newDevice" {
		device_state, err := service.addDevice(tx.Account, tx.Symbol, tx.Description, tx.DeviceTags)
		if err != nil {
			return nil, err
		}
		r.States = append(r.States, device_state)
	}

	return &block.TransactionWithData{
		Transaction: t,
		Receipt:     r,
	}, nil
}
