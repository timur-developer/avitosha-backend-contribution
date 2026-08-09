package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/repository/postgres"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestGameRepositoryBootstrapsProfileAndKeepsEventIdempotent(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	user, err := userRepository.Create(context.Background(), usecase.CreateUserParams{
		Email: "game-owner@example.com", PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repository := postgres.NewGameRepository(pool)
	txManager := postgres.NewTxManager(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	petID := uuid.New()
	storyID := uuid.New()

	err = txManager.WithinTx(context.Background(), func(ctx context.Context) error {
		_, txErr := repository.GetOrCreateGamePet(ctx, model.Pet{
			ID: petID, UserID: user.ID, Name: "Авитоша", Level: 1,
			Mood: model.PetMoodCalm, CreatedAt: now, UpdatedAt: now,
		})
		if txErr != nil {
			return txErr
		}
		_, txErr = repository.GetOrCreateStoryProgress(ctx, model.UserStoryProgress{
			ID: storyID, UserID: user.ID, StoryCode: "FIRST_ROOM", Status: model.StoryStatusActive,
			StartedAt: now, CreatedAt: now, UpdatedAt: now,
		})
		if txErr != nil {
			return txErr
		}
		placedAt := now
		if txErr = repository.EnsureInitialRoomItem(ctx, model.UserRoomItem{
			ID: uuid.New(), UserID: user.ID, ItemCode: "BOX", Status: model.RoomItemStatusPlaced,
			UnlockedAt: now, PlacedAt: &placedAt, CreatedAt: now, UpdatedAt: now,
		}); txErr != nil {
			return txErr
		}
		_, txErr = repository.AssignStoryTask(ctx, user.ID, "FIRST_ROOM", 1, now)
		return txErr
	})
	if err != nil {
		t.Fatalf("bootstrap game profile: %v", err)
	}

	tasks, err := repository.ListTaskProgress(context.Background(), user.ID)
	if err != nil || len(tasks) != 1 || tasks[0].Task.Code != "VIEW_FURNITURE_ADS" {
		t.Fatalf("tasks = %+v, error = %v", tasks, err)
	}
	room, err := repository.ListRoomItems(context.Background(), user.ID)
	if err != nil || len(room) == 0 || room[0].Status != model.RoomItemStatusPlaced {
		t.Fatalf("room = %+v, error = %v", room, err)
	}

	eventID := uuid.New()
	actionCandidate := model.UserAction{
		ID: uuid.New(), UserID: user.ID, EventID: eventID, ActionType: model.ActionTypeAdViewed,
		Category: stringPointer("FURNITURE"), Metadata: json.RawMessage(`{}`), OccurredAt: now, CreatedAt: now,
	}
	first, inserted, err := repository.InsertAction(context.Background(), actionCandidate)
	if err != nil || !inserted {
		t.Fatalf("first InsertAction() inserted = %v, error = %v", inserted, err)
	}
	if err := repository.CompleteAction(context.Background(), first.ID, now, []model.DomainEvent{}); err != nil {
		t.Fatalf("CompleteAction() error = %v", err)
	}
	second, inserted, err := repository.InsertAction(context.Background(), actionCandidate)
	if err != nil || inserted || second.ID != first.ID || second.ProcessedAt == nil {
		t.Fatalf("idempotent action = %+v, inserted = %v, error = %v", second, inserted, err)
	}
}

func TestGameTransactionRollsBackInsertedAction(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	user, err := userRepository.Create(context.Background(), usecase.CreateUserParams{
		Email: "rollback-owner@example.com", PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repository := postgres.NewGameRepository(pool)
	txManager := postgres.NewTxManager(pool)
	eventID := uuid.New()
	sentinel := errors.New("force rollback")
	err = txManager.WithinTx(context.Background(), func(ctx context.Context) error {
		_, inserted, insertErr := repository.InsertAction(ctx, model.UserAction{
			ID: uuid.New(), UserID: user.ID, EventID: eventID, ActionType: model.ActionTypeAdViewed,
			Metadata: json.RawMessage(`{}`), OccurredAt: time.Now().UTC(), CreatedAt: time.Now().UTC(),
		})
		if insertErr != nil {
			return insertErr
		}
		if !inserted {
			t.Fatal("new action was not inserted")
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("transaction error = %v, want sentinel", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM user_actions WHERE event_id = $1`, eventID).Scan(&count); err != nil {
		t.Fatalf("count rolled back action: %v", err)
	}
	if count != 0 {
		t.Fatalf("rolled back action count = %d, want 0", count)
	}
}

func TestConcurrentSameEventIsProcessedOnce(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	user, err := userRepository.Create(context.Background(), usecase.CreateUserParams{
		Email: "concurrent-event@example.com", PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	service := usecase.NewGameService(usecase.GameServiceDependencies{
		Repository: postgres.NewGameRepository(pool), TxManager: postgres.NewTxManager(pool), IDGenerator: uuid.New,
	})
	now := time.Now().UTC().Truncate(time.Microsecond)
	category := "FURNITURE"
	command := usecase.ProcessActionCommand{
		UserID: user.ID, EventID: uuid.New(), ActionType: model.ActionTypeAdViewed,
		EntityID: stringPointer("advert-concurrent"), Category: &category,
		Metadata: json.RawMessage(`{}`), OccurredAt: now, Now: now,
	}

	start := make(chan struct{})
	results := make(chan usecase.ProcessActionResult, 2)
	errorsChannel := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			result, processErr := service.ProcessAction(context.Background(), command)
			results <- result
			errorsChannel <- processErr
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)
	close(errorsChannel)
	for processErr := range errorsChannel {
		if processErr != nil {
			t.Fatalf("ProcessAction() error = %v", processErr)
		}
	}
	duplicates := 0
	for result := range results {
		if result.Duplicate {
			duplicates++
		}
	}
	if duplicates != 1 {
		t.Fatalf("duplicate results = %d, want 1", duplicates)
	}

	var actions, dailyActions, taskProgress int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM user_actions WHERE event_id = $1`, command.EventID).Scan(&actions); err != nil {
		t.Fatalf("count actions: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT actions_count FROM daily_progress WHERE user_id = $1`, user.ID).Scan(&dailyActions); err != nil {
		t.Fatalf("read daily actions: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
SELECT ut.progress FROM user_tasks ut JOIN tasks t ON t.id = ut.task_id
WHERE ut.user_id = $1 AND t.code = 'VIEW_FURNITURE_ADS'
`, user.ID).Scan(&taskProgress); err != nil {
		t.Fatalf("read task progress: %v", err)
	}
	if actions != 1 || dailyActions != 1 || taskProgress != 1 {
		t.Fatalf("actions = %d, daily = %d, task progress = %d", actions, dailyActions, taskProgress)
	}
}

func TestConcurrentTaskCompletionRewardsOnce(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	user, err := userRepository.Create(context.Background(), usecase.CreateUserParams{
		Email: "concurrent-completion@example.com", PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	repository := postgres.NewGameRepository(pool)
	service := usecase.NewGameService(usecase.GameServiceDependencies{
		Repository: repository, TxManager: postgres.NewTxManager(pool), IDGenerator: uuid.New,
	})
	now := time.Now().UTC().Truncate(time.Microsecond)
	category := "FURNITURE"
	for index := 0; index < 4; index++ {
		_, err := service.ProcessAction(context.Background(), usecase.ProcessActionCommand{
			UserID: user.ID, EventID: uuid.New(), ActionType: model.ActionTypeAdViewed,
			EntityID: stringPointer(uuid.NewString()), Category: &category, Metadata: json.RawMessage(`{}`),
			OccurredAt: now.Add(time.Duration(index) * time.Second), Now: now.Add(time.Duration(index) * time.Second),
		})
		if err != nil {
			t.Fatalf("seed progress %d: %v", index+1, err)
		}
	}

	start := make(chan struct{})
	errorsChannel := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for index := 0; index < 2; index++ {
		waitGroup.Add(1)
		go func(offset int) {
			defer waitGroup.Done()
			<-start
			_, processErr := service.ProcessAction(context.Background(), usecase.ProcessActionCommand{
				UserID: user.ID, EventID: uuid.New(), ActionType: model.ActionTypeAdViewed,
				EntityID: stringPointer(uuid.NewString()), Category: &category, Metadata: json.RawMessage(`{}`),
				OccurredAt: now.Add(time.Duration(10+offset) * time.Second), Now: now.Add(time.Duration(10+offset) * time.Second),
			})
			errorsChannel <- processErr
		}(index)
	}
	close(start)
	waitGroup.Wait()
	close(errorsChannel)
	for processErr := range errorsChannel {
		if processErr != nil {
			t.Fatalf("concurrent completion error = %v", processErr)
		}
	}

	pet, err := repository.GetGamePetByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get pet: %v", err)
	}
	story, err := repository.GetStorySnapshot(context.Background(), user.ID, usecase.FirstRoomStoryCode)
	if err != nil {
		t.Fatalf("get story: %v", err)
	}
	weekly, err := repository.GetWeeklyPosition(context.Background(), user.ID, usecase.WeekStart(now))
	if err != nil {
		t.Fatalf("get weekly progress: %v", err)
	}
	balances, err := repository.ListRewardBalances(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list reward balances: %v", err)
	}
	var rewardTransactions int
	if err := pool.QueryRow(context.Background(), `
SELECT COUNT(*) FROM reward_transactions WHERE user_id = $1
`, user.ID).Scan(&rewardTransactions); err != nil {
		t.Fatalf("count reward transactions: %v", err)
	}
	if pet.GrowthXP != 30 || story.Progress.CurrentStage != 1 || weekly.Score != 100 || weekly.CompletedTasks != 1 ||
		len(balances) != 1 || balances[0].Balance != 10 || balances[0].EarnedTotal != 10 || rewardTransactions != 1 {
		t.Fatalf("pet XP = %d, story = %d, weekly = %+v, balances = %+v, reward transactions = %d",
			pet.GrowthXP, story.Progress.CurrentStage, weekly, balances, rewardTransactions)
	}
}

func TestFirstActionCreditsStreakRewardWithoutSQLTypeConflict(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	user, err := userRepository.Create(context.Background(), usecase.CreateUserParams{
		Email: "streak-reward@example.com", PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repository := postgres.NewGameRepository(pool)
	service := usecase.NewGameService(usecase.GameServiceDependencies{
		Repository: repository, TxManager: postgres.NewTxManager(pool), IDGenerator: uuid.New,
	})
	now := time.Now().UTC().Truncate(time.Microsecond)
	category := "FURNITURE"

	result, err := service.ProcessAction(context.Background(), usecase.ProcessActionCommand{
		UserID: user.ID, EventID: uuid.New(), ActionType: model.ActionTypeAdViewed,
		EntityID: stringPointer("advert-first"), Category: &category, Metadata: json.RawMessage(`{}`),
		OccurredAt: now, Now: now,
	})
	if err != nil {
		t.Fatalf("ProcessAction() error = %v", err)
	}
	if result.Duplicate {
		t.Fatal("first action was unexpectedly marked as duplicate")
	}

	balances, err := repository.ListRewardBalances(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list reward balances: %v", err)
	}
	if len(balances) != 1 || balances[0].Balance != 2 || balances[0].EarnedTotal != 2 {
		t.Fatalf("reward balances = %+v, want streak reward balance 2", balances)
	}
}

func TestGameRepositoryRoomUnlockIsIdempotent(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	user, err := userRepository.Create(context.Background(), usecase.CreateUserParams{
		Email: "room-owner@example.com", PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repository := postgres.NewGameRepository(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	placedAt := now
	item := model.UserRoomItem{
		ID: uuid.New(), UserID: user.ID, ItemCode: "DESK", Status: model.RoomItemStatusPlaced,
		UnlockedAt: now, PlacedAt: &placedAt, CreatedAt: now, UpdatedAt: now,
	}
	first, err := repository.UnlockRoomItem(context.Background(), item)
	if err != nil || !first {
		t.Fatalf("first UnlockRoomItem() unlocked = %v, error = %v", first, err)
	}
	item.ID = uuid.New()
	second, err := repository.UnlockRoomItem(context.Background(), item)
	if err != nil || second {
		t.Fatalf("second UnlockRoomItem() unlocked = %v, error = %v", second, err)
	}
}
