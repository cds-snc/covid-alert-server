package timemath

import (
	"time"

	pb "github.com/CovidShield/server/pkg/proto/covidshield"
)

const (
	SecondsInHour  = 3600
	SecondsInDay   = 86400
	HoursInDay     = 24
)

func HourNumber(t time.Time) uint32 {
	return uint32(t.Unix() / SecondsInHour)
}

func DateNumber(t time.Time) uint32 {
	return uint32(t.Unix() / SecondsInDay)
}

func MostRecentUTCMidnight(t time.Time) time.Time {
	return time.Unix((t.UTC().Unix()/SecondsInDay)*SecondsInDay, 0).UTC()
}

func HourNumberAtStartOfDate(dateNumber uint32) uint32 {
	return dateNumber * HoursInDay
}

func HourNumberPlusDays(hourNumber uint32, days int) uint32 {
	return uint32(int(hourNumber) + HoursInDay*days)
}

func RollingStartIntervalNumberPlusDays(rsin int32, days int) int32 {
	return int32(int(rsin) + days*pb.MaxTEKRollingPeriod)
}

func CurrentDateNumber() uint32 {
	return DateNumber(time.Now())
}
