package service

import (
	"github.com/escalopa/minichain/explorer/internal/core/domain"
	"github.com/escalopa/minichain/explorer/internal/core/port"
)

// Explorer is the query side of the core: everything the driving
// adapters (HTTP, future CLI) can ask about the chain.
type Explorer struct {
	repo port.ChainRepository
}

func NewExplorer(repo port.ChainRepository) *Explorer {
	return &Explorer{repo: repo}
}

func (e *Explorer) Height() int {
	return e.repo.Height()
}

func (e *Explorer) Recent(limit int) []domain.Block {
	return e.repo.Recent(limit)
}

func (e *Explorer) Block(ref string) (domain.Block, bool) {
	return e.repo.ByRef(ref)
}

func (e *Explorer) Address(addr string) (uint64, []domain.TxRef) {
	return domain.AddressInfo(e.repo.Snapshot(), addr)
}
