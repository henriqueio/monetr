package fixtures

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/monetr/monetr/server/internal/testutils"
	"github.com/monetr/monetr/server/models"
	"github.com/monetr/monetr/server/repository"
	"github.com/monetr/monetr/server/util"
	"github.com/stretchr/testify/require"
)

func GivenIHaveAFundingSchedule(
	t *testing.T,
	clock clock.Clock,
	bankAccount *models.BankAccount,
	ruleString string,
	excludeWeekends bool,
) *models.FundingSchedule {
	require.NotNil(t, bankAccount, "must provide a valid bank account")
	require.NotZero(t, bankAccount.BankAccountId, "bank account must have a valid Id")
	require.NotZero(t, bankAccount.AccountId, "bank account must have a valid account Id")
	require.NotZero(t, bankAccount.Link.CreatedBy, "bank account must have a valid created by user Id")

	if excludeWeekends {
		panic("sorry I haven't implemented this yet")
	}

	log := testutils.GetLog(t)
	db := testutils.GetPgDatabase(t)
	repo := repository.NewRepositoryFromSession(
		clock,
		bankAccount.Link.CreatedBy,
		bankAccount.AccountId,
		db,
		log,
	)

	timezone := testutils.MustEz(t, bankAccount.Account.GetTimezone)
	rule := testutils.RuleToSet(t, timezone, ruleString, clock.Now())
	nextOccurrence := util.Midnight(rule.After(clock.Now(), false), timezone)

	fundingSchedule := models.FundingSchedule{
		AccountId:              bankAccount.AccountId,
		Account:                bankAccount.Account,
		BankAccountId:          bankAccount.BankAccountId,
		BankAccount:            bankAccount,
		Name:                   gofakeit.Generate("Payday {uuid}"),
		Description:            gofakeit.Generate("{sentence:5}"),
		RuleSet:                rule,
		ExcludeWeekends:        excludeWeekends,
		LastRecurrence:         nil,
		NextRecurrence:         nextOccurrence,
		NextRecurrenceOriginal: nextOccurrence,
	}

	require.NoError(t, repo.CreateFundingSchedule(
		t.Context(),
		&fundingSchedule,
	), "must be able to create funding schedule")

	return &fundingSchedule
}
