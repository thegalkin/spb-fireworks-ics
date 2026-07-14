// Package calendar содержит чистые функции над датами для СПб-календаря.
package calendar

import "time"

// LastSaturdayOfJune возвращает последнюю субботу июня в указанном году.
// Используется для «Алых парусов»: праздник стабильно попадает в этот день с 2017.
func LastSaturdayOfJune(year int) time.Time {
	d := time.Date(year, time.July, 1, 0, 0, 0, 0, time.UTC)
	for d.Weekday() != time.Saturday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

// LastSundayOfJuly возвращает последнее воскресенье июля.
// «День ВМФ» с 2025 года — последнее воскресенье июля (ФЗ-419 от 25.12.2023).
func LastSundayOfJuly(year int) time.Time {
	d := time.Date(year, time.July, 31, 0, 0, 0, 0, time.UTC)
	for d.Weekday() != time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}
