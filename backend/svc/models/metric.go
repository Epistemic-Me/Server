package models

import "errors"

type Metric struct {
	Label       string
	Numerator   int32
	Denominator int32
}

func (m Metric) ToPercentage() float64 {
	if m.Denominator == 0 {
		return 0
	}
	return float64(m.Numerator) / float64(m.Denominator)
}

func Average(metrics []Metric) (Metric, error) {
	if len(metrics) == 0 {
		return Metric{}, errors.New("no metrics provided")
	}

	originalLabel := metrics[0].Label
	var totalNumerator int32 = 0
	var totalDenominator int32 = 0

	for _, metric := range metrics {
		if metric.Label != originalLabel {
			return Metric{}, errors.New("all metrics must have the same label")
		}
		totalNumerator += metric.Numerator
		totalDenominator += metric.Denominator
	}

	// Construct the average metric
	averageMetric := Metric{
		Label:       originalLabel,
		Numerator:   totalNumerator,
		Denominator: totalDenominator,
	}

	return averageMetric, nil
}
