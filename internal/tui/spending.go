package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"subscription-tracker/internal/app"
	"subscription-tracker/internal/db"
)

type SpendingView struct {
	month         int
	year          int
	cutoffDay     int
	periodStart   time.Time
	periodEnd     time.Time
	monthlyTotal  float64
	yearlyTotal   float64
	monthlySubs   []db.Subscription
	yearlySubs    []db.Subscription
	monthlySalary float64
	remaining     float64
	loading       bool
	err           error
}

func NewSpendingView() *SpendingView {
	now := time.Now()
	return &SpendingView{
		month:   int(now.Month()),
		year:    now.Year(),
		loading: true,
	}
}

func (v *SpendingView) Init(a *app.App) tea.Cmd {
	return v.loadSpending(a)
}

func (v *SpendingView) loadSpending(a *app.App) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		summary, err := a.SpendingService.CalculateForMonth(ctx, v.year, v.month)
		if err != nil {
			return spendingErrMsg{err}
		}

		return spendingLoadedMsg{
			monthlySubs:   summary.MonthlyItems,
			yearlySubs:    summary.YearlyItems,
			monthlyTotal:  summary.MonthlyTotal,
			yearlyTotal:   summary.YearlyTotal,
			cutoffDay:     summary.CutoffDay,
			periodStart:   summary.PeriodStart,
			periodEnd:     summary.PeriodEnd,
			monthlySalary: summary.MonthlySalary,
			remaining:     summary.Remaining,
		}
	}
}

type spendingLoadedMsg struct {
	monthlySubs   []db.Subscription
	yearlySubs    []db.Subscription
	monthlyTotal  float64
	yearlyTotal   float64
	cutoffDay     int
	periodStart   time.Time
	periodEnd     time.Time
	monthlySalary float64
	remaining     float64
}

type spendingErrMsg struct {
	err error
}

func (v *SpendingView) Update(msg tea.Msg, a *app.App) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			v.month--
			if v.month < 1 {
				v.month = 12
				v.year--
			}
			v.loading = true
			return false, v.loadSpending(a)
		case "right", "l":
			v.month++
			if v.month > 12 {
				v.month = 1
				v.year++
			}
			v.loading = true
			return false, v.loadSpending(a)
		case "q", "esc":
			return true, nil
		}
	case spendingLoadedMsg:
		v.loading = false
		v.monthlySubs = msg.monthlySubs
		v.yearlySubs = msg.yearlySubs
		v.monthlyTotal = msg.monthlyTotal
		v.yearlyTotal = msg.yearlyTotal
		v.cutoffDay = msg.cutoffDay
		v.periodStart = msg.periodStart
		v.periodEnd = msg.periodEnd
		v.monthlySalary = msg.monthlySalary
		v.remaining = msg.remaining
		return false, nil
	case spendingErrMsg:
		v.loading = false
		v.err = msg.err
		return false, nil
	}
	return false, nil
}

func (v *SpendingView) View() string {
	var b strings.Builder

	monthName := time.Month(v.month).String()
	title := fmt.Sprintf("Spending for %s %d", monthName, v.year)
	b.WriteString(TitleStyle.Render(title) + "\n")

	// Show date range
	if !v.periodStart.IsZero() && !v.periodEnd.IsZero() {
		dateRange := fmt.Sprintf("%s - %s",
			v.periodStart.Format("Jan 2, 2006"),
			v.periodEnd.Format("Jan 2, 2006"))
		b.WriteString(SubtitleStyle.Render(dateRange) + "\n")
	}
	b.WriteString("\n")

	if v.loading {
		b.WriteString("Loading...\n")
		return BoxStyle.Render(b.String())
	}

	if v.err != nil {
		b.WriteString(ErrorStyle.Render("Error: "+v.err.Error()) + "\n\n")
	}

	// Monthly subscriptions
	if len(v.monthlySubs) > 0 {
		b.WriteString(SubtitleStyle.Render("Monthly Subscriptions:") + "\n")
		for _, s := range v.monthlySubs {
			b.WriteString(fmt.Sprintf("  %s: %.2f %s\n", s.Name, s.Amount, s.Currency))
		}
		b.WriteString(fmt.Sprintf("  %s\n\n", AmountStyle.Render(fmt.Sprintf("Subtotal: %.2f", v.monthlyTotal))))
	}

	// Yearly subscriptions renewing this period
	if len(v.yearlySubs) > 0 {
		b.WriteString(YearlyStyle.Render("Yearly Subscriptions Renewing This Period:") + "\n")
		for _, s := range v.yearlySubs {
			renewal := ""
			if s.NextRenewalDate.Valid {
				renewal = s.NextRenewalDate.String
			}
			b.WriteString(fmt.Sprintf("  %s: %.2f %s (renews %s)\n", s.Name, s.Amount, s.Currency, renewal))
		}
		b.WriteString(fmt.Sprintf("  %s\n\n", AmountStyle.Render(fmt.Sprintf("Subtotal: %.2f", v.yearlyTotal))))
	}

	if len(v.monthlySubs) == 0 && len(v.yearlySubs) == 0 {
		b.WriteString(SubtitleStyle.Render("No subscriptions for this period.") + "\n\n")
	}

	// Total
	total := v.monthlyTotal + v.yearlyTotal
	b.WriteString("────────────────────────────────\n")
	b.WriteString(AmountStyle.Render(fmt.Sprintf("TOTAL SUBSCRIPTIONS: %.2f", total)) + "\n")

	if v.yearlyTotal > 0 {
		avgMonthly := v.monthlyTotal + (v.yearlyTotal / 12)
		b.WriteString(SubtitleStyle.Render(fmt.Sprintf("Average Monthly (yearly prorated): %.2f", avgMonthly)) + "\n")
	}

	// Show remaining money if salary is configured
	if v.monthlySalary > 0 {
		b.WriteString("\n")
		b.WriteString(SubtitleStyle.Render(fmt.Sprintf("Monthly Salary: %.2f", v.monthlySalary)) + "\n")
		if v.remaining >= 0 {
			b.WriteString(SuccessStyle.Render(fmt.Sprintf("REMAINING: %.2f", v.remaining)) + "\n")
		} else {
			b.WriteString(ErrorStyle.Render(fmt.Sprintf("OVER BUDGET: %.2f", -v.remaining)) + "\n")
		}
	}

	b.WriteString("\n" + HelpStyle.Render("[←/→] change month  [q/esc] back"))

	return BoxStyle.Render(b.String())
}
