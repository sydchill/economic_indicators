package macro

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// MacroSnapshot struct
type MacroSnapshot struct {
	Country             string  `json:"Country"`
	GDPGrowthRate       float64 `json:"GDP Growth Rate"`
	UnemploymentRate    float64 `json:"Unemployment Rate"`
	InflationRate       float64 `json:"Inflation Rate"`
	InterestRate        float64 `json:"Interest Rate"`
	InflationRateMoM    float64 `json:"Inflation Rate MoM "`
	BalanceOfTrade      float64 `json:"Balance of Trade "`
	CurrentAccount      float64 `json:"Current Account"`
	BusinessConfidence  float64 `json:"Business Confidence "`
	ManufacturingPMI    float64 `json:"Manufacturing PMI"`
	ServicesPMIRaw      any     `json:"Services PMI"` // can be "" or number
	ConsumerConfidence  float64 `json:"Consumer Confidence "`
	RetailSalesMoM      float64 `json:"Retail Sales MoM "`
	GDPAnnualGrowthRate float64 `json:"GDP Annual Growth Rate"`
}

// ParsedServicesPMI tries to read Services PMI even if it's "" in JSON.
func (m MacroSnapshot) ParsedServicesPMI() *float64 {
	switch v := m.ServicesPMIRaw.(type) {
	case float64:
		return &v
	case string:
		if v == "" {
			return nil
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return &f
		}
	}
	return nil
}

// LoadSnapshots func
func LoadSnapshots(path string) ([]MacroSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read macro file: %w", err)
	}

	var snapshots []MacroSnapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return nil, fmt.Errorf("unmarshal macro file: %w", err)
	}

	return snapshots, nil
}
