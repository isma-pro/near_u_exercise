package domain

// Account holds an investor's cash and fund positions.
type Account struct {
	ID             string
	Cash           Money
	ReservedCash   Money
	Positions      map[string]Units // fund_id -> held units
	ReservedUnits  map[string]Units // fund_id -> reserved units for pending redemptions
}

// AvailableCash returns cash minus reserved cash.
func (a Account) AvailableCash() Money {
	return a.Cash.Sub(a.ReservedCash)
}

// AvailableUnits returns held units minus reserved units for a fund.
func (a Account) AvailableUnits(fundID string) Units {
	return a.Positions[fundID].Sub(a.ReservedUnits[fundID])
}
