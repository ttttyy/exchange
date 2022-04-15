package bexchange

import (
	"fmt"
	rbt "github.com/psampaz/gods/trees/redblacktree"
	"sync"

	"trader/utils"
)

const MAX_PRICE = 10000000

type PricePoint struct {
	orderHead *Order
	orderTail *Order
}
type commitPrice struct {
	Price  float64
	Amount uint32
}

func (pp *PricePoint) Insert(o *Order) {
	if pp.orderHead == nil {
		pp.orderHead = o
		pp.orderTail = o
	} else {
		pp.orderTail.Next = o
		pp.orderTail = o
	}
}

type OrderStatus int

const (
	OS_NEW OrderStatus = iota
	OS_OPEN
	OS_PARTIAL
	OS_FILLED
	OS_CANCELLED
)

type Order struct {
	Id     uint64
	IsBuy  bool
	Price  float64
	Amount uint32
	Status OrderStatus
	Next   *Order
}

func (o *Order) String() string {
	return fmt.Sprintf("\nOrder{id:%v,isBuy:%v,price:%v,amount:%v}",
		o.Id, o.IsBuy, o.Price, o.Amount)
}

func NewOrder(Id uint64, isBuy bool, price float64, amount uint32) *Order {
	return &Order{Id: Id, IsBuy: isBuy, Price: price, Amount: amount,
		Status: OS_NEW}
}

type OrderBook struct {
	mu sync.Mutex
	// These are more estimates than reportable values
	ask        float64 //委卖价
	bid        float64 //委买价
	FinalPrice float64
	//orderIndex map[uint64]*Order
	//prices     [MAX_PRICE]*PricePoint
	orderBuyTree  rbt.Tree
	orderSellTree rbt.Tree
	actions       chan<- *Action
}

func NewOrderBook(actions chan<- *Action) *OrderBook {
	ob := new(OrderBook)
	ob.bid = 0
	ob.ask = MAX_PRICE
	ob.FinalPrice = 0
	/*for i := range ob.prices {
		ob.prices[i] = new(PricePoint)
	}
	ob.actions = actions
	ob.orderIndex = make(map[uint64]*Order)*/
	ob.orderBuyTree = rbt.Tree{Comparator: utils.Comparator_buy}
	ob.orderSellTree = rbt.Tree{Comparator: utils.Comparator_sell}
	ob.actions = actions
	return ob
}

func (ob *OrderBook) AddOrder(o *Order) {
	// Try to fill immediately
	if o.IsBuy {
		ob.actions <- NewBuyAction(o)
		ob.FillBuy(o)
	} else {
		ob.actions <- NewSellAction(o)
		ob.FillSell(o)
	}

	// Into the book
	if o.Amount > 0 {
		ob.openOrder(o)
	}
}

func (ob *OrderBook) openOrder(o *Order) {
	//pp := ob.prices[o.price]
	//pp.Insert(o)
	if o.IsBuy {
		ob.orderBuyTree.Put(utils.Combined_key{o.Id, o.Price}, o)
	} else {
		ob.orderSellTree.Put(utils.Combined_key{o.Id, o.Price}, o)
	}
	o.Status = OS_OPEN
	if o.IsBuy && o.Price > ob.bid {
		ob.bid = o.Price
	} else if !o.IsBuy && o.Price < ob.ask {
		ob.ask = o.Price
	}
	//ob.orderIndex[o.id] = o
}

func (ob *OrderBook) FillBuy(o *Order) {
	for ob.ask < o.Price && o.Amount > 0 {
		/*
			pp := ob.prices[ob.ask]
			ppOrderHead := pp.orderHead
			for ppOrderHead != nil {
				ob.fill(o, ppOrderHead)
				ppOrderHead = ppOrderHead.next
				pp.orderHead = ppOrderHead
			}*/
		iter := ob.orderSellTree.Iterator()
		for iter.Begin(); iter.Next(); {
			it := iter.Value().(*Order)
			if it.Status != OS_CANCELLED && it.Price <= o.Price {
				ob.fill(o, it)
			} else {
				break
			}
		}
	}
}

func (ob *OrderBook) FillSell(o *Order) {
	for ob.bid >= o.Price && o.Amount > 0 {
		/*pp := ob.prices[ob.bid]
		ppOrderHead := pp.orderHead
		for ppOrderHead != nil {
			ob.fill(o, ppOrderHead)
			ppOrderHead = ppOrderHead.next
			pp.orderHead = ppOrderHead
		}
		ob.bid--*/
		iter := ob.orderBuyTree.Iterator()
		for iter.Begin(); iter.Next(); {
			it := iter.Value().(*Order)
			if it.Status != OS_CANCELLED && it.Price >= o.Price {
				ob.fill(o, it)
			} else {
				break
			}
		}
	}
}

func (ob *OrderBook) fill(o, ppOrderHead *Order) {
	if ppOrderHead.Amount >= o.Amount {
		ob.actions <- NewFilledAction(o, ppOrderHead)
		ppOrderHead.Amount -= o.Amount
		o.Amount = 0
		o.Status = OS_FILLED
		return
	} else {
		// Partial fill
		if ppOrderHead.Amount > 0 {
			ob.actions <- NewPartialFilledAction(o, ppOrderHead)
			o.Amount -= ppOrderHead.Amount
			o.Status = OS_PARTIAL
			ppOrderHead.Amount = 0
		}
	}
	ob.FinalPrice = ppOrderHead.Price
}

func (ob *OrderBook) CancelOrder(Id uint64, price float64, buy bool) {
	ob.actions <- NewCancelAction(Id)
	//if o, ok := ob.orderIndex[id]; ok {
	//	// If this is the last order at a particular price point
	//	// we need to update the bid/ask...right? Maybe not?
	//	o.Amount = 0
	//	o.Status = OS_CANCELLED
	//}
	if buy {
		it, found := ob.orderBuyTree.Get(utils.Combined_key{Id, price})
		if found {
			o := it.(*Order)
			o.Amount = 0
			o.Status = OS_CANCELLED
		}
	}
	ob.actions <- NewCancelledAction(Id)
}

func (ob *OrderBook) Done() {
	ob.actions <- NewDoneAction()
}

func (ob *OrderBook) BuyPrice5() []commitPrice {
	ob.mu.Lock()
	res := make([]commitPrice, 0)
	iter := ob.orderBuyTree.Iterator()
	i := 0
	for iter.Begin(); i < 5 && iter.Next(); {
		i++
		it := iter.Value().(*Order)
		res = append(res, commitPrice{it.Price, it.Amount})
	}
	ob.mu.Unlock()
	return res
}
func (ob *OrderBook) SellPrice5() []commitPrice {
	ob.mu.Lock()
	res := make([]commitPrice, 5)
	iter := ob.orderSellTree.Iterator()
	i := 0
	for iter.Begin(); i < 5 && iter.Next(); {
		i++
		it := iter.Value().(*Order)
		res = append(res, commitPrice{it.Price, it.Amount})
	}
	ob.mu.Unlock()
	return res
}
func (ob *OrderBook) GetFinalPrice() float64 {
	ob.mu.Lock()
	x := ob.FinalPrice
	ob.mu.Unlock()
	return x
}
