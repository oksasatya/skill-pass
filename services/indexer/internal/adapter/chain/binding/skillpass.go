// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package binding

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// SkillPassCertificateCertificate is an auto generated low-level Go binding around an user-defined struct.
type SkillPassCertificateCertificate struct {
	Title         string
	RecipientName string
	IssuerName    string
	Description   string
	MetadataURI   string
	IssuedAt      *big.Int
}

// SkillPassCertificateMetaData contains all meta data concerning the SkillPassCertificate contract.
var SkillPassCertificateMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"initialOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"approve\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getApproved\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getCertificate\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"cert\",\"type\":\"tuple\",\"internalType\":\"structSkillPassCertificate.Certificate\",\"components\":[{\"name\":\"title\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"recipientName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"issuerName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"description\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"metadataURI\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"issuedAt\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isApprovedForAll\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"issueCertificate\",\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"title\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"recipientName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"issuerName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"description\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"metadataURI\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"locked\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"name\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"ownerOf\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"safeTransferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"safeTransferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setApprovalForAll\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"supportsInterface\",\"inputs\":[{\"name\":\"interfaceId\",\"type\":\"bytes4\",\"internalType\":\"bytes4\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"symbol\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"tokenURI\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Approval\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"approved\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ApprovalForAll\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"operator\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"approved\",\"type\":\"bool\",\"indexed\":false,\"internalType\":\"bool\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"CertificateIssued\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"recipient\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"title\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"},{\"name\":\"issuerName\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"},{\"name\":\"issuedAt\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Locked\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Transfer\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ApprovalDisabled\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ERC721IncorrectOwner\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721InsufficientApproval\",\"inputs\":[{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ERC721InvalidApprover\",\"inputs\":[{\"name\":\"approver\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721InvalidOperator\",\"inputs\":[{\"name\":\"operator\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721InvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721InvalidReceiver\",\"inputs\":[{\"name\":\"receiver\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721InvalidSender\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC721NonexistentToken\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"OwnableInvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"OwnableUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"Soulbound\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"StringTooLong\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ZeroRecipient\",\"inputs\":[]}]",
}

// SkillPassCertificateABI is the input ABI used to generate the binding from.
// Deprecated: Use SkillPassCertificateMetaData.ABI instead.
var SkillPassCertificateABI = SkillPassCertificateMetaData.ABI

// SkillPassCertificate is an auto generated Go binding around an Ethereum contract.
type SkillPassCertificate struct {
	SkillPassCertificateCaller     // Read-only binding to the contract
	SkillPassCertificateTransactor // Write-only binding to the contract
	SkillPassCertificateFilterer   // Log filterer for contract events
}

// SkillPassCertificateCaller is an auto generated read-only Go binding around an Ethereum contract.
type SkillPassCertificateCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SkillPassCertificateTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SkillPassCertificateTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SkillPassCertificateFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SkillPassCertificateFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SkillPassCertificateSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SkillPassCertificateSession struct {
	Contract     *SkillPassCertificate // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// SkillPassCertificateCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SkillPassCertificateCallerSession struct {
	Contract *SkillPassCertificateCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// SkillPassCertificateTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SkillPassCertificateTransactorSession struct {
	Contract     *SkillPassCertificateTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// SkillPassCertificateRaw is an auto generated low-level Go binding around an Ethereum contract.
type SkillPassCertificateRaw struct {
	Contract *SkillPassCertificate // Generic contract binding to access the raw methods on
}

// SkillPassCertificateCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SkillPassCertificateCallerRaw struct {
	Contract *SkillPassCertificateCaller // Generic read-only contract binding to access the raw methods on
}

// SkillPassCertificateTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SkillPassCertificateTransactorRaw struct {
	Contract *SkillPassCertificateTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSkillPassCertificate creates a new instance of SkillPassCertificate, bound to a specific deployed contract.
func NewSkillPassCertificate(address common.Address, backend bind.ContractBackend) (*SkillPassCertificate, error) {
	contract, err := bindSkillPassCertificate(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SkillPassCertificate{SkillPassCertificateCaller: SkillPassCertificateCaller{contract: contract}, SkillPassCertificateTransactor: SkillPassCertificateTransactor{contract: contract}, SkillPassCertificateFilterer: SkillPassCertificateFilterer{contract: contract}}, nil
}

// NewSkillPassCertificateCaller creates a new read-only instance of SkillPassCertificate, bound to a specific deployed contract.
func NewSkillPassCertificateCaller(address common.Address, caller bind.ContractCaller) (*SkillPassCertificateCaller, error) {
	contract, err := bindSkillPassCertificate(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SkillPassCertificateCaller{contract: contract}, nil
}

// NewSkillPassCertificateTransactor creates a new write-only instance of SkillPassCertificate, bound to a specific deployed contract.
func NewSkillPassCertificateTransactor(address common.Address, transactor bind.ContractTransactor) (*SkillPassCertificateTransactor, error) {
	contract, err := bindSkillPassCertificate(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SkillPassCertificateTransactor{contract: contract}, nil
}

// NewSkillPassCertificateFilterer creates a new log filterer instance of SkillPassCertificate, bound to a specific deployed contract.
func NewSkillPassCertificateFilterer(address common.Address, filterer bind.ContractFilterer) (*SkillPassCertificateFilterer, error) {
	contract, err := bindSkillPassCertificate(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SkillPassCertificateFilterer{contract: contract}, nil
}

// bindSkillPassCertificate binds a generic wrapper to an already deployed contract.
func bindSkillPassCertificate(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SkillPassCertificateMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SkillPassCertificate *SkillPassCertificateRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SkillPassCertificate.Contract.SkillPassCertificateCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SkillPassCertificate *SkillPassCertificateRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.SkillPassCertificateTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SkillPassCertificate *SkillPassCertificateRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.SkillPassCertificateTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SkillPassCertificate *SkillPassCertificateCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SkillPassCertificate.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SkillPassCertificate *SkillPassCertificateTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SkillPassCertificate *SkillPassCertificateTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.contract.Transact(opts, method, params...)
}

// Approve is a free data retrieval call binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address , uint256 ) pure returns()
func (_SkillPassCertificate *SkillPassCertificateCaller) Approve(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) error {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "approve", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// Approve is a free data retrieval call binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address , uint256 ) pure returns()
func (_SkillPassCertificate *SkillPassCertificateSession) Approve(arg0 common.Address, arg1 *big.Int) error {
	return _SkillPassCertificate.Contract.Approve(&_SkillPassCertificate.CallOpts, arg0, arg1)
}

// Approve is a free data retrieval call binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address , uint256 ) pure returns()
func (_SkillPassCertificate *SkillPassCertificateCallerSession) Approve(arg0 common.Address, arg1 *big.Int) error {
	return _SkillPassCertificate.Contract.Approve(&_SkillPassCertificate.CallOpts, arg0, arg1)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_SkillPassCertificate *SkillPassCertificateCaller) BalanceOf(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "balanceOf", owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_SkillPassCertificate *SkillPassCertificateSession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _SkillPassCertificate.Contract.BalanceOf(&_SkillPassCertificate.CallOpts, owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _SkillPassCertificate.Contract.BalanceOf(&_SkillPassCertificate.CallOpts, owner)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_SkillPassCertificate *SkillPassCertificateCaller) GetApproved(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "getApproved", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_SkillPassCertificate *SkillPassCertificateSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _SkillPassCertificate.Contract.GetApproved(&_SkillPassCertificate.CallOpts, tokenId)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 tokenId) view returns(address)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) GetApproved(tokenId *big.Int) (common.Address, error) {
	return _SkillPassCertificate.Contract.GetApproved(&_SkillPassCertificate.CallOpts, tokenId)
}

// GetCertificate is a free data retrieval call binding the contract method 0x51640fee.
//
// Solidity: function getCertificate(uint256 tokenId) view returns((string,string,string,string,string,uint256) cert, address recipient)
func (_SkillPassCertificate *SkillPassCertificateCaller) GetCertificate(opts *bind.CallOpts, tokenId *big.Int) (struct {
	Cert      SkillPassCertificateCertificate
	Recipient common.Address
}, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "getCertificate", tokenId)

	outstruct := new(struct {
		Cert      SkillPassCertificateCertificate
		Recipient common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Cert = *abi.ConvertType(out[0], new(SkillPassCertificateCertificate)).(*SkillPassCertificateCertificate)
	outstruct.Recipient = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// GetCertificate is a free data retrieval call binding the contract method 0x51640fee.
//
// Solidity: function getCertificate(uint256 tokenId) view returns((string,string,string,string,string,uint256) cert, address recipient)
func (_SkillPassCertificate *SkillPassCertificateSession) GetCertificate(tokenId *big.Int) (struct {
	Cert      SkillPassCertificateCertificate
	Recipient common.Address
}, error) {
	return _SkillPassCertificate.Contract.GetCertificate(&_SkillPassCertificate.CallOpts, tokenId)
}

// GetCertificate is a free data retrieval call binding the contract method 0x51640fee.
//
// Solidity: function getCertificate(uint256 tokenId) view returns((string,string,string,string,string,uint256) cert, address recipient)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) GetCertificate(tokenId *big.Int) (struct {
	Cert      SkillPassCertificateCertificate
	Recipient common.Address
}, error) {
	return _SkillPassCertificate.Contract.GetCertificate(&_SkillPassCertificate.CallOpts, tokenId)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_SkillPassCertificate *SkillPassCertificateCaller) IsApprovedForAll(opts *bind.CallOpts, owner common.Address, operator common.Address) (bool, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "isApprovedForAll", owner, operator)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_SkillPassCertificate *SkillPassCertificateSession) IsApprovedForAll(owner common.Address, operator common.Address) (bool, error) {
	return _SkillPassCertificate.Contract.IsApprovedForAll(&_SkillPassCertificate.CallOpts, owner, operator)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address owner, address operator) view returns(bool)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) IsApprovedForAll(owner common.Address, operator common.Address) (bool, error) {
	return _SkillPassCertificate.Contract.IsApprovedForAll(&_SkillPassCertificate.CallOpts, owner, operator)
}

// Locked is a free data retrieval call binding the contract method 0xb45a3c0e.
//
// Solidity: function locked(uint256 tokenId) view returns(bool)
func (_SkillPassCertificate *SkillPassCertificateCaller) Locked(opts *bind.CallOpts, tokenId *big.Int) (bool, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "locked", tokenId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Locked is a free data retrieval call binding the contract method 0xb45a3c0e.
//
// Solidity: function locked(uint256 tokenId) view returns(bool)
func (_SkillPassCertificate *SkillPassCertificateSession) Locked(tokenId *big.Int) (bool, error) {
	return _SkillPassCertificate.Contract.Locked(&_SkillPassCertificate.CallOpts, tokenId)
}

// Locked is a free data retrieval call binding the contract method 0xb45a3c0e.
//
// Solidity: function locked(uint256 tokenId) view returns(bool)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) Locked(tokenId *big.Int) (bool, error) {
	return _SkillPassCertificate.Contract.Locked(&_SkillPassCertificate.CallOpts, tokenId)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_SkillPassCertificate *SkillPassCertificateCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_SkillPassCertificate *SkillPassCertificateSession) Name() (string, error) {
	return _SkillPassCertificate.Contract.Name(&_SkillPassCertificate.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) Name() (string, error) {
	return _SkillPassCertificate.Contract.Name(&_SkillPassCertificate.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SkillPassCertificate *SkillPassCertificateCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SkillPassCertificate *SkillPassCertificateSession) Owner() (common.Address, error) {
	return _SkillPassCertificate.Contract.Owner(&_SkillPassCertificate.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) Owner() (common.Address, error) {
	return _SkillPassCertificate.Contract.Owner(&_SkillPassCertificate.CallOpts)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_SkillPassCertificate *SkillPassCertificateCaller) OwnerOf(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "ownerOf", tokenId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_SkillPassCertificate *SkillPassCertificateSession) OwnerOf(tokenId *big.Int) (common.Address, error) {
	return _SkillPassCertificate.Contract.OwnerOf(&_SkillPassCertificate.CallOpts, tokenId)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 tokenId) view returns(address)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) OwnerOf(tokenId *big.Int) (common.Address, error) {
	return _SkillPassCertificate.Contract.OwnerOf(&_SkillPassCertificate.CallOpts, tokenId)
}

// SetApprovalForAll is a free data retrieval call binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address , bool ) pure returns()
func (_SkillPassCertificate *SkillPassCertificateCaller) SetApprovalForAll(opts *bind.CallOpts, arg0 common.Address, arg1 bool) error {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "setApprovalForAll", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// SetApprovalForAll is a free data retrieval call binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address , bool ) pure returns()
func (_SkillPassCertificate *SkillPassCertificateSession) SetApprovalForAll(arg0 common.Address, arg1 bool) error {
	return _SkillPassCertificate.Contract.SetApprovalForAll(&_SkillPassCertificate.CallOpts, arg0, arg1)
}

// SetApprovalForAll is a free data retrieval call binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address , bool ) pure returns()
func (_SkillPassCertificate *SkillPassCertificateCallerSession) SetApprovalForAll(arg0 common.Address, arg1 bool) error {
	return _SkillPassCertificate.Contract.SetApprovalForAll(&_SkillPassCertificate.CallOpts, arg0, arg1)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_SkillPassCertificate *SkillPassCertificateCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_SkillPassCertificate *SkillPassCertificateSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _SkillPassCertificate.Contract.SupportsInterface(&_SkillPassCertificate.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _SkillPassCertificate.Contract.SupportsInterface(&_SkillPassCertificate.CallOpts, interfaceId)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_SkillPassCertificate *SkillPassCertificateCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_SkillPassCertificate *SkillPassCertificateSession) Symbol() (string, error) {
	return _SkillPassCertificate.Contract.Symbol(&_SkillPassCertificate.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) Symbol() (string, error) {
	return _SkillPassCertificate.Contract.Symbol(&_SkillPassCertificate.CallOpts)
}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_SkillPassCertificate *SkillPassCertificateCaller) TokenURI(opts *bind.CallOpts, tokenId *big.Int) (string, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "tokenURI", tokenId)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_SkillPassCertificate *SkillPassCertificateSession) TokenURI(tokenId *big.Int) (string, error) {
	return _SkillPassCertificate.Contract.TokenURI(&_SkillPassCertificate.CallOpts, tokenId)
}

// TokenURI is a free data retrieval call binding the contract method 0xc87b56dd.
//
// Solidity: function tokenURI(uint256 tokenId) view returns(string)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) TokenURI(tokenId *big.Int) (string, error) {
	return _SkillPassCertificate.Contract.TokenURI(&_SkillPassCertificate.CallOpts, tokenId)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_SkillPassCertificate *SkillPassCertificateCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SkillPassCertificate.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_SkillPassCertificate *SkillPassCertificateSession) TotalSupply() (*big.Int, error) {
	return _SkillPassCertificate.Contract.TotalSupply(&_SkillPassCertificate.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_SkillPassCertificate *SkillPassCertificateCallerSession) TotalSupply() (*big.Int, error) {
	return _SkillPassCertificate.Contract.TotalSupply(&_SkillPassCertificate.CallOpts)
}

// IssueCertificate is a paid mutator transaction binding the contract method 0xde8c98fd.
//
// Solidity: function issueCertificate(address recipient, string title, string recipientName, string issuerName, string description, string metadataURI) returns(uint256 tokenId)
func (_SkillPassCertificate *SkillPassCertificateTransactor) IssueCertificate(opts *bind.TransactOpts, recipient common.Address, title string, recipientName string, issuerName string, description string, metadataURI string) (*types.Transaction, error) {
	return _SkillPassCertificate.contract.Transact(opts, "issueCertificate", recipient, title, recipientName, issuerName, description, metadataURI)
}

// IssueCertificate is a paid mutator transaction binding the contract method 0xde8c98fd.
//
// Solidity: function issueCertificate(address recipient, string title, string recipientName, string issuerName, string description, string metadataURI) returns(uint256 tokenId)
func (_SkillPassCertificate *SkillPassCertificateSession) IssueCertificate(recipient common.Address, title string, recipientName string, issuerName string, description string, metadataURI string) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.IssueCertificate(&_SkillPassCertificate.TransactOpts, recipient, title, recipientName, issuerName, description, metadataURI)
}

// IssueCertificate is a paid mutator transaction binding the contract method 0xde8c98fd.
//
// Solidity: function issueCertificate(address recipient, string title, string recipientName, string issuerName, string description, string metadataURI) returns(uint256 tokenId)
func (_SkillPassCertificate *SkillPassCertificateTransactorSession) IssueCertificate(recipient common.Address, title string, recipientName string, issuerName string, description string, metadataURI string) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.IssueCertificate(&_SkillPassCertificate.TransactOpts, recipient, title, recipientName, issuerName, description, metadataURI)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_SkillPassCertificate *SkillPassCertificateTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SkillPassCertificate.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_SkillPassCertificate *SkillPassCertificateSession) RenounceOwnership() (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.RenounceOwnership(&_SkillPassCertificate.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_SkillPassCertificate *SkillPassCertificateTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.RenounceOwnership(&_SkillPassCertificate.TransactOpts)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_SkillPassCertificate *SkillPassCertificateTransactor) SafeTransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _SkillPassCertificate.contract.Transact(opts, "safeTransferFrom", from, to, tokenId)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_SkillPassCertificate *SkillPassCertificateSession) SafeTransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.SafeTransferFrom(&_SkillPassCertificate.TransactOpts, from, to, tokenId)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId) returns()
func (_SkillPassCertificate *SkillPassCertificateTransactorSession) SafeTransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.SafeTransferFrom(&_SkillPassCertificate.TransactOpts, from, to, tokenId)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_SkillPassCertificate *SkillPassCertificateTransactor) SafeTransferFrom0(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _SkillPassCertificate.contract.Transact(opts, "safeTransferFrom0", from, to, tokenId, data)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_SkillPassCertificate *SkillPassCertificateSession) SafeTransferFrom0(from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.SafeTransferFrom0(&_SkillPassCertificate.TransactOpts, from, to, tokenId, data)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 tokenId, bytes data) returns()
func (_SkillPassCertificate *SkillPassCertificateTransactorSession) SafeTransferFrom0(from common.Address, to common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.SafeTransferFrom0(&_SkillPassCertificate.TransactOpts, from, to, tokenId, data)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_SkillPassCertificate *SkillPassCertificateTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _SkillPassCertificate.contract.Transact(opts, "transferFrom", from, to, tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_SkillPassCertificate *SkillPassCertificateSession) TransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.TransferFrom(&_SkillPassCertificate.TransactOpts, from, to, tokenId)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 tokenId) returns()
func (_SkillPassCertificate *SkillPassCertificateTransactorSession) TransferFrom(from common.Address, to common.Address, tokenId *big.Int) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.TransferFrom(&_SkillPassCertificate.TransactOpts, from, to, tokenId)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_SkillPassCertificate *SkillPassCertificateTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _SkillPassCertificate.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_SkillPassCertificate *SkillPassCertificateSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.TransferOwnership(&_SkillPassCertificate.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_SkillPassCertificate *SkillPassCertificateTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _SkillPassCertificate.Contract.TransferOwnership(&_SkillPassCertificate.TransactOpts, newOwner)
}

// SkillPassCertificateApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the SkillPassCertificate contract.
type SkillPassCertificateApprovalIterator struct {
	Event *SkillPassCertificateApproval // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SkillPassCertificateApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SkillPassCertificateApproval)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SkillPassCertificateApproval)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SkillPassCertificateApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SkillPassCertificateApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SkillPassCertificateApproval represents a Approval event raised by the SkillPassCertificate contract.
type SkillPassCertificateApproval struct {
	Owner    common.Address
	Approved common.Address
	TokenId  *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_SkillPassCertificate *SkillPassCertificateFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, approved []common.Address, tokenId []*big.Int) (*SkillPassCertificateApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var approvedRule []interface{}
	for _, approvedItem := range approved {
		approvedRule = append(approvedRule, approvedItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _SkillPassCertificate.contract.FilterLogs(opts, "Approval", ownerRule, approvedRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &SkillPassCertificateApprovalIterator{contract: _SkillPassCertificate.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_SkillPassCertificate *SkillPassCertificateFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *SkillPassCertificateApproval, owner []common.Address, approved []common.Address, tokenId []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var approvedRule []interface{}
	for _, approvedItem := range approved {
		approvedRule = append(approvedRule, approvedItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _SkillPassCertificate.contract.WatchLogs(opts, "Approval", ownerRule, approvedRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SkillPassCertificateApproval)
				if err := _SkillPassCertificate.contract.UnpackLog(event, "Approval", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)
func (_SkillPassCertificate *SkillPassCertificateFilterer) ParseApproval(log types.Log) (*SkillPassCertificateApproval, error) {
	event := new(SkillPassCertificateApproval)
	if err := _SkillPassCertificate.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SkillPassCertificateApprovalForAllIterator is returned from FilterApprovalForAll and is used to iterate over the raw logs and unpacked data for ApprovalForAll events raised by the SkillPassCertificate contract.
type SkillPassCertificateApprovalForAllIterator struct {
	Event *SkillPassCertificateApprovalForAll // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SkillPassCertificateApprovalForAllIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SkillPassCertificateApprovalForAll)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SkillPassCertificateApprovalForAll)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SkillPassCertificateApprovalForAllIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SkillPassCertificateApprovalForAllIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SkillPassCertificateApprovalForAll represents a ApprovalForAll event raised by the SkillPassCertificate contract.
type SkillPassCertificateApprovalForAll struct {
	Owner    common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAll is a free log retrieval operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_SkillPassCertificate *SkillPassCertificateFilterer) FilterApprovalForAll(opts *bind.FilterOpts, owner []common.Address, operator []common.Address) (*SkillPassCertificateApprovalForAllIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _SkillPassCertificate.contract.FilterLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &SkillPassCertificateApprovalForAllIterator{contract: _SkillPassCertificate.contract, event: "ApprovalForAll", logs: logs, sub: sub}, nil
}

// WatchApprovalForAll is a free log subscription operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_SkillPassCertificate *SkillPassCertificateFilterer) WatchApprovalForAll(opts *bind.WatchOpts, sink chan<- *SkillPassCertificateApprovalForAll, owner []common.Address, operator []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _SkillPassCertificate.contract.WatchLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SkillPassCertificateApprovalForAll)
				if err := _SkillPassCertificate.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApprovalForAll is a log parse operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_SkillPassCertificate *SkillPassCertificateFilterer) ParseApprovalForAll(log types.Log) (*SkillPassCertificateApprovalForAll, error) {
	event := new(SkillPassCertificateApprovalForAll)
	if err := _SkillPassCertificate.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SkillPassCertificateCertificateIssuedIterator is returned from FilterCertificateIssued and is used to iterate over the raw logs and unpacked data for CertificateIssued events raised by the SkillPassCertificate contract.
type SkillPassCertificateCertificateIssuedIterator struct {
	Event *SkillPassCertificateCertificateIssued // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SkillPassCertificateCertificateIssuedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SkillPassCertificateCertificateIssued)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SkillPassCertificateCertificateIssued)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SkillPassCertificateCertificateIssuedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SkillPassCertificateCertificateIssuedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SkillPassCertificateCertificateIssued represents a CertificateIssued event raised by the SkillPassCertificate contract.
type SkillPassCertificateCertificateIssued struct {
	TokenId    *big.Int
	Recipient  common.Address
	Title      string
	IssuerName string
	IssuedAt   *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterCertificateIssued is a free log retrieval operation binding the contract event 0x69c2ef69279bd95361b92e81471ac7e6062d893b90c8da20820cfd23255b96aa.
//
// Solidity: event CertificateIssued(uint256 indexed tokenId, address indexed recipient, string title, string issuerName, uint256 issuedAt)
func (_SkillPassCertificate *SkillPassCertificateFilterer) FilterCertificateIssued(opts *bind.FilterOpts, tokenId []*big.Int, recipient []common.Address) (*SkillPassCertificateCertificateIssuedIterator, error) {

	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _SkillPassCertificate.contract.FilterLogs(opts, "CertificateIssued", tokenIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &SkillPassCertificateCertificateIssuedIterator{contract: _SkillPassCertificate.contract, event: "CertificateIssued", logs: logs, sub: sub}, nil
}

// WatchCertificateIssued is a free log subscription operation binding the contract event 0x69c2ef69279bd95361b92e81471ac7e6062d893b90c8da20820cfd23255b96aa.
//
// Solidity: event CertificateIssued(uint256 indexed tokenId, address indexed recipient, string title, string issuerName, uint256 issuedAt)
func (_SkillPassCertificate *SkillPassCertificateFilterer) WatchCertificateIssued(opts *bind.WatchOpts, sink chan<- *SkillPassCertificateCertificateIssued, tokenId []*big.Int, recipient []common.Address) (event.Subscription, error) {

	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _SkillPassCertificate.contract.WatchLogs(opts, "CertificateIssued", tokenIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SkillPassCertificateCertificateIssued)
				if err := _SkillPassCertificate.contract.UnpackLog(event, "CertificateIssued", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCertificateIssued is a log parse operation binding the contract event 0x69c2ef69279bd95361b92e81471ac7e6062d893b90c8da20820cfd23255b96aa.
//
// Solidity: event CertificateIssued(uint256 indexed tokenId, address indexed recipient, string title, string issuerName, uint256 issuedAt)
func (_SkillPassCertificate *SkillPassCertificateFilterer) ParseCertificateIssued(log types.Log) (*SkillPassCertificateCertificateIssued, error) {
	event := new(SkillPassCertificateCertificateIssued)
	if err := _SkillPassCertificate.contract.UnpackLog(event, "CertificateIssued", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SkillPassCertificateLockedIterator is returned from FilterLocked and is used to iterate over the raw logs and unpacked data for Locked events raised by the SkillPassCertificate contract.
type SkillPassCertificateLockedIterator struct {
	Event *SkillPassCertificateLocked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SkillPassCertificateLockedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SkillPassCertificateLocked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SkillPassCertificateLocked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SkillPassCertificateLockedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SkillPassCertificateLockedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SkillPassCertificateLocked represents a Locked event raised by the SkillPassCertificate contract.
type SkillPassCertificateLocked struct {
	TokenId *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterLocked is a free log retrieval operation binding the contract event 0x032bc66be43dbccb7487781d168eb7bda224628a3b2c3388bdf69b532a3a1611.
//
// Solidity: event Locked(uint256 tokenId)
func (_SkillPassCertificate *SkillPassCertificateFilterer) FilterLocked(opts *bind.FilterOpts) (*SkillPassCertificateLockedIterator, error) {

	logs, sub, err := _SkillPassCertificate.contract.FilterLogs(opts, "Locked")
	if err != nil {
		return nil, err
	}
	return &SkillPassCertificateLockedIterator{contract: _SkillPassCertificate.contract, event: "Locked", logs: logs, sub: sub}, nil
}

// WatchLocked is a free log subscription operation binding the contract event 0x032bc66be43dbccb7487781d168eb7bda224628a3b2c3388bdf69b532a3a1611.
//
// Solidity: event Locked(uint256 tokenId)
func (_SkillPassCertificate *SkillPassCertificateFilterer) WatchLocked(opts *bind.WatchOpts, sink chan<- *SkillPassCertificateLocked) (event.Subscription, error) {

	logs, sub, err := _SkillPassCertificate.contract.WatchLogs(opts, "Locked")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SkillPassCertificateLocked)
				if err := _SkillPassCertificate.contract.UnpackLog(event, "Locked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseLocked is a log parse operation binding the contract event 0x032bc66be43dbccb7487781d168eb7bda224628a3b2c3388bdf69b532a3a1611.
//
// Solidity: event Locked(uint256 tokenId)
func (_SkillPassCertificate *SkillPassCertificateFilterer) ParseLocked(log types.Log) (*SkillPassCertificateLocked, error) {
	event := new(SkillPassCertificateLocked)
	if err := _SkillPassCertificate.contract.UnpackLog(event, "Locked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SkillPassCertificateOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the SkillPassCertificate contract.
type SkillPassCertificateOwnershipTransferredIterator struct {
	Event *SkillPassCertificateOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SkillPassCertificateOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SkillPassCertificateOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SkillPassCertificateOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SkillPassCertificateOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SkillPassCertificateOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SkillPassCertificateOwnershipTransferred represents a OwnershipTransferred event raised by the SkillPassCertificate contract.
type SkillPassCertificateOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_SkillPassCertificate *SkillPassCertificateFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*SkillPassCertificateOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _SkillPassCertificate.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &SkillPassCertificateOwnershipTransferredIterator{contract: _SkillPassCertificate.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_SkillPassCertificate *SkillPassCertificateFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *SkillPassCertificateOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _SkillPassCertificate.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SkillPassCertificateOwnershipTransferred)
				if err := _SkillPassCertificate.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_SkillPassCertificate *SkillPassCertificateFilterer) ParseOwnershipTransferred(log types.Log) (*SkillPassCertificateOwnershipTransferred, error) {
	event := new(SkillPassCertificateOwnershipTransferred)
	if err := _SkillPassCertificate.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SkillPassCertificateTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the SkillPassCertificate contract.
type SkillPassCertificateTransferIterator struct {
	Event *SkillPassCertificateTransfer // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SkillPassCertificateTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SkillPassCertificateTransfer)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SkillPassCertificateTransfer)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SkillPassCertificateTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SkillPassCertificateTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SkillPassCertificateTransfer represents a Transfer event raised by the SkillPassCertificate contract.
type SkillPassCertificateTransfer struct {
	From    common.Address
	To      common.Address
	TokenId *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_SkillPassCertificate *SkillPassCertificateFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address, tokenId []*big.Int) (*SkillPassCertificateTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _SkillPassCertificate.contract.FilterLogs(opts, "Transfer", fromRule, toRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &SkillPassCertificateTransferIterator{contract: _SkillPassCertificate.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_SkillPassCertificate *SkillPassCertificateFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *SkillPassCertificateTransfer, from []common.Address, to []common.Address, tokenId []*big.Int) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _SkillPassCertificate.contract.WatchLogs(opts, "Transfer", fromRule, toRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SkillPassCertificateTransfer)
				if err := _SkillPassCertificate.contract.UnpackLog(event, "Transfer", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)
func (_SkillPassCertificate *SkillPassCertificateFilterer) ParseTransfer(log types.Log) (*SkillPassCertificateTransfer, error) {
	event := new(SkillPassCertificateTransfer)
	if err := _SkillPassCertificate.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
