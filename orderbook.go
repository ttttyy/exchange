package main

const MAX_PRICE = 10000000

type PricePoint struct {
    orderHead *Order
    orderTail *Order
}

func (pp *PricePoint) Insert(o *Order) {
    if pp.orderHead == nil {
        pp.orderHead = o
        pp.orderTail = o
    } else {
        pp.orderTail.next = o
        pp.orderTail = o
    }
}

func (pp *PricePoint) Peek() *Order {
    return pp.orderHead
}

func (pp *PricePoint) Pop() *Order {
    order := pp.orderHead
    pp.orderHead = order.next
    // Pop the last node
    if order.next == nil {
        pp.orderTail = nil
    }
    return order
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
    id uint64
    isBuy bool
    price uint32
    amount uint32
    filled uint32
    status OrderStatus
    next *Order
}

func NewOrder(id uint64, isBuy bool, price uint32, amount uint32) *Order {
    o := new(Order)
    o.id = id
    o.isBuy = isBuy
    o.price = price
    o.amount = amount
    o.status = OS_NEW
    return o
}

type OrderBook struct {
    // These are more estimates than reportable values
    ask uint32
    bid uint32
    orderIndex map[uint64]*Order
    prices [MAX_PRICE]*PricePoint
    actions chan<- *Action
}

func NewOrderBook(actions chan<- *Action) *OrderBook {
    ob := new(OrderBook)
    ob.bid = 0
    ob.ask = MAX_PRICE
    for i := range ob.prices {
        ob.prices[i] = new(PricePoint)
    }
    ob.actions = actions
    ob.orderIndex = make(map[uint64]*Order)
    return ob
}

func (ob *OrderBook) Price() uint32 {
    return (ob.ask - ob.bid) / 2
}

func (ob *OrderBook) AddOrder(o *Order) {
    // Try to fill immediately
    if o.isBuy {
        ob.actions <- NewBuyAction(o)
        ob.FillBuy(o)
    } else {
        ob.actions <- NewSellAction(o)
        ob.FillSell(o)
    }

    // Into the book
    if o.amount > 0 {
        ob.openOrder(o)
    }
}

func (ob *OrderBook) openOrder(o *Order) {
    pp := ob.prices[o.price]
    pp.Insert(o)
    o.status = OS_OPEN
    if o.isBuy && o.price > ob.bid {
        ob.bid = o.price
    } else if !o.isBuy && o.price < ob.ask {
        ob.ask = o.price
    }
    ob.orderIndex[o.id] = o
}

func (ob *OrderBook) FillBuy(o *Order) {
    for ob.ask < o.price && o.amount > 0 {
        pp := ob.prices[ob.ask]
        ppOrderHead := pp.orderHead
        for ppOrderHead != nil {
            if ppOrderHead.amount >= o.amount {
                ob.actions <- NewFilledAction(o, ppOrderHead, ob.ask)
                ppOrderHead.amount -= o.amount
                o.amount = 0
                o.status = OS_FILLED
                return
            } else {
                // Partial fill
                if ppOrderHead.amount > 0 {
                    ob.actions <- NewPartialFilledAction(o, ppOrderHead, ppOrderHead.amount, ob.ask)
                    o.amount -= ppOrderHead.amount
                    o.status = OS_PARTIAL
                    ppOrderHead.amount = 0
                }

                ppOrderHead = ppOrderHead.next
                pp.orderHead = ppOrderHead
            }
        }
        ob.ask++
    }
}

func (ob *OrderBook) FillSell(o *Order) {
    for ob.bid >= o.price && o.amount > 0 {
        pp := ob.prices[ob.bid]
        ppOrderHead := pp.orderHead
        for ppOrderHead != nil {
            if ppOrderHead.amount >= o.amount {
                ob.actions <- NewFilledAction(o, ppOrderHead, ob.bid)
                ppOrderHead.amount -= o.amount
                o.amount = 0
                o.status = OS_FILLED
            } else {
                // Partial fill
                if ppOrderHead.amount > 0 {
                    ob.actions <- NewPartialFilledAction(o, ppOrderHead, ppOrderHead.amount, ob.bid)
                    ppOrderHead.amount = 0
                    o.amount -= ppOrderHead.amount
                    o.status = OS_PARTIAL
                }

                ppOrderHead = ppOrderHead.next
                pp.orderHead = ppOrderHead
            }
        }
        ob.bid--
    }
}

func (ob *OrderBook) CancelOrder(id uint64) {
    ob.actions <- NewCancelAction(id)
    if o, ok := ob.orderIndex[id]; ok {
        // If this is the last order at a particular price point
        // we need to update the bid/ask...right? Maybe not?
        o.amount = 0
        o.status = OS_CANCELLED
    }
    ob.actions <- NewCancelledAction(id)
}
