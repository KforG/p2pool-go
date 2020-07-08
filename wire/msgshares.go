package wire

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"log"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	btcwire "github.com/btcsuite/btcd/wire"
	"github.com/gertjaap/p2pool-go/logging"
	p2pnet "github.com/gertjaap/p2pool-go/net"
	"github.com/gertjaap/p2pool-go/util"
)

var _ P2PoolMessage = &MsgShares{}

type MsgShares struct {
	Shares []Share
}

type Share struct {
	Type           uint64
	MinHeader      SmallBlockHeader
	ShareInfo      ShareInfo
	RefMerkleLink  []*chainhash.Hash
	LastTxOutNonce uint64
	HashLink       HashLink
	MerkleLink     []*chainhash.Hash
	GenTXHash      *chainhash.Hash
	MerkleRoot     *chainhash.Hash
	RefHash        *chainhash.Hash
	Hash           *chainhash.Hash
}

type HashLink struct {
	State  string
	Length uint64
}

type SmallBlockHeader struct {
	Version       int32
	PreviousBlock *chainhash.Hash
	Timestamp     uint32
	Bits          uint32
	Nonce         uint32
}

type Ref struct {
	Identifier string
	ShareInfo  ShareInfo
}

type ShareInfo struct {
	ShareData            ShareData
	SegwitData           SegwitData
	NewTransactionHashes []*chainhash.Hash
	TransactionHashRefs  []TransactionHashRef
	FarShareHash         *chainhash.Hash
	MaxBits              int32
	Bits                 int32
	Timestamp            int32
	AbsHeight            int32
	AbsWork              *big.Int
}

type TransactionHashRef struct {
	ShareCount uint64
	TxCount    uint64
}

type ShareData struct {
	PreviousShareHash *chainhash.Hash
	CoinBase          string
	Nonce             uint32
	PubKeyHash        []byte
	PubKeyHashVersion uint8
	Subsidy           uint64
	Donation          uint16
	StaleInfo         StaleInfo
	DesiredVersion    uint64
}

type StaleInfo uint8

const (
	StaleInfoNone   = StaleInfo(0)
	StaleInfoOrphan = StaleInfo(253)
	StaleInfoDOA    = StaleInfo(254)
)

var DonationScript []byte
var GenTxBeforeRefHash []byte

type SegwitData struct {
	TXIDMerkleLink  []*chainhash.Hash
	WTXIDMerkleRoot *chainhash.Hash
}

func GetRefHash(n p2pnet.Network, si ShareInfo, refMerkleLink []*chainhash.Hash, segwit bool) (*chainhash.Hash, error) {
	r := Ref{
		Identifier: string(n.Identifier),
		ShareInfo:  si,
	}
	var buf bytes.Buffer

	err := WriteRef(&buf, r, segwit)
	if err != nil {
		return nil, err
	}

	logging.Debugf("get_ref_hash_check_merkle_link_hash_input: %x", buf.Bytes())
	tip, _ := chainhash.NewHash(util.Sha256d(buf.Bytes()))
	logging.Debugf("get_ref_hash_check_merkle_link_input: %x", tip.CloneBytes())
	return CalcMerkleLink(tip, refMerkleLink, 0)
}

func CalcMerkleLink(tip *chainhash.Hash, link []*chainhash.Hash, linkIndex int) (*chainhash.Hash, error) {
	link = append(link, tip)
	if len(link) == 1 {
		return link[0], nil
	}
	h := link[0]
	for i := 1; i < len(link); i++ {
		hashBytes := make([]byte, 64)
		hIdx, nIdx := 0, 32
		if linkIndex>>i&1 == 1 {
			nIdx, hIdx = 0, 32
		}
		copy(hashBytes[hIdx:], h.CloneBytes())
		copy(hashBytes[nIdx:], link[i].CloneBytes())
		h, _ = chainhash.NewHash(util.Sha256d(hashBytes))
	}
	return h, nil
}

func CalcHashLink(hl HashLink, data []byte, ending []byte) (*chainhash.Hash, error) {

	extralength := hl.Length % 64
	extra := ending[len(ending)-int(extralength):]

	s := util.NewSha256()
	h := s.CalcMidState(data, []byte(hl.State), extra, hl.Length)
	s.Reset()
	s.Write(h[:])
	return chainhash.NewHash(s.Sum(nil))
}

func ReadShares(r io.Reader) ([]Share, error) {
	shares := make([]Share, 0)
	count, err := ReadVarInt(r)
	if err != nil {
		return shares, err
	}
	log.Printf("Deserializing %d shares", count)
	for i := uint64(0); i < count; i++ {
		s := Share{}
		s.Type, err = ReadVarInt(r)
		if err != nil {
			return shares, err
		}

		log.Printf("Type is %d", s.Type)

		// REad length - not needed for us
		len, err := ReadVarInt(r)
		if err != nil {
			return shares, err
		}
		log.Printf("Share message length is %d", len)

		s.MinHeader, err = ReadSmallBlockHeader(r)
		if err != nil {
			return shares, err
		}

		log.Printf("Minheader is Prevblock: %s, Version: %d, Timestamp: %d, Bits: %d, Nonce: %d", s.MinHeader.PreviousBlock.String(), s.MinHeader.Version, s.MinHeader.Timestamp, s.MinHeader.Bits, s.MinHeader.Nonce)

		s.ShareInfo, err = ReadShareInfo(r, s.Type >= 17)
		if err != nil {
			return shares, err
		}

		log.Printf("Read shareinfo. MaxBits %d, Bits %d, AbsHeight %d, AbsWork: %x", s.ShareInfo.MaxBits, s.ShareInfo.Bits, s.ShareInfo.AbsHeight, s.ShareInfo.AbsWork.Bytes())

		s.RefMerkleLink, err = ReadChainHashList(r)
		if err != nil {
			return shares, err
		}

		err = binary.Read(r, binary.LittleEndian, &s.LastTxOutNonce)
		if err != nil {
			return shares, err
		}

		log.Printf("Read lasttxoutnonce: %d", s.LastTxOutNonce)

		s.HashLink, err = ReadHashLink(r)
		if err != nil {
			return shares, err
		}

		s.MerkleLink, err = ReadChainHashList(r)
		if err != nil {
			return shares, err
		}

		s.RefHash, _ = GetRefHash(p2pnet.ActiveNetwork, s.ShareInfo, s.RefMerkleLink, s.Type >= 17)

		var buf bytes.Buffer
		buf.Write(s.RefHash.CloneBytes())
		binary.Write(&buf, binary.LittleEndian, s.LastTxOutNonce)
		binary.Write(&buf, binary.LittleEndian, int32(0))
		logging.Debugf("GenTxBeforeRefHash: %x", GenTxBeforeRefHash)
		s.GenTXHash, err = CalcHashLink(s.HashLink, buf.Bytes(), GenTxBeforeRefHash)
		if err != nil {
			return shares, err
		}

		merkleLink := s.MerkleLink
		if s.Type >= 17 {
			merkleLink = s.ShareInfo.SegwitData.TXIDMerkleLink
		}
		s.MerkleRoot, err = CalcMerkleLink(s.GenTXHash, merkleLink, 0)
		if err != nil {
			return shares, err
		}

		buf.Reset()

		hdr := btcwire.NewBlockHeader(s.MinHeader.Version, s.MinHeader.PreviousBlock, s.MerkleRoot, s.MinHeader.Bits, s.MinHeader.Nonce)
		hdr.Timestamp = time.Unix(int64(s.MinHeader.Timestamp), 0)
		hdr.Serialize(&buf)

		logging.Debugf("Header: %x", buf.Bytes())

		s.Hash, _ = chainhash.NewHash(util.Sha256d(buf.Bytes()))

		shares = append(shares, s)
	}
	return shares, nil
}

func (m *MsgShares) FromBytes(b []byte) error {
	var err error

	r := bytes.NewReader(b)
	m.Shares, err = ReadShares(r)
	if err != nil {
		return err
	}
	log.Printf("Deserialized %d shares", len(m.Shares))
	return nil
}

func (m *MsgShares) ToBytes() ([]byte, error) {
	var buf bytes.Buffer

	err := WriteVarInt(&buf, uint64(len(m.Shares)))
	if err != nil {
		return nil, err
	}
	for _, s := range m.Shares {
		err = WriteVarInt(&buf, s.Type)
		if err != nil {
			return nil, err
		}
		err = WriteSmallBlockHeader(&buf, s.MinHeader)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (m *MsgShares) Command() string {
	return "shares"
}

func init() {
	DonationScript, _ = hex.DecodeString("410418a74130b2f4fad899d8ed2bff272bc43a03c8ca72897ae3da584d7a770b5a9ea8dd1b37a620d27c6cf6d5a7a9bbd6872f5981e95816d701d94f201c5d093be6ac")
	GenTxBeforeRefHash = make([]byte, len(DonationScript)+12)
	copy(GenTxBeforeRefHash, []byte{byte(len(DonationScript))})
	copy(GenTxBeforeRefHash[1:], DonationScript)
	copy(GenTxBeforeRefHash[len(DonationScript)+9:], []byte{42, 0x6A, 0x28})

}
