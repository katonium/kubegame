package main

import (
	"context"
	"net/http"

	"github.com/katonium/kubegame/backend/domain/repository"
	"github.com/katonium/kubegame/backend/domain/service"
	"github.com/katonium/kubegame/backend/infrastructure/kubernetes"
	infraRepo "github.com/katonium/kubegame/backend/infrastructure/repository"
	"github.com/katonium/kubegame/backend/infrastructure/websocket"
	"github.com/katonium/kubegame/backend/usecase"
	"github.com/katonium/kubegame/backend/util/logger"
	"go.uber.org/fx"
	"k8s.io/client-go/kubernetes/fake"
)

func main() {
	app := fx.New(
		// Provide repositories
		fx.Provide(
			fx.Annotate(
				infraRepo.NewMemoryGameRepository,
				fx.As(new(repository.GameRepository)),
			),
			fx.Annotate(
				infraRepo.NewMemoryPodRepository,
				fx.As(new(repository.PodRepository)),
			),
			fx.Annotate(
				infraRepo.NewMemoryNodeRepository,
				fx.As(new(repository.NodeRepository)),
			),
		),

		// Provide services
		fx.Provide(
			fx.Annotate(
				kubernetes.NewKubernetesService,
				fx.As(new(service.KubernetesService)),
			),
			fx.Annotate(
				func() service.SchedulerService {
					client := fake.NewSimpleClientset()
					return kubernetes.NewSchedulerService(client)
				},
				fx.As(new(service.SchedulerService)),
			),
			fx.Annotate(
				websocket.NewWebSocketService,
				fx.As(new(service.WebSocketService)),
			),
		),

		// Provide use cases
		fx.Provide(
			usecase.NewGameEngineUseCase,
			usecase.NewGameUseCase,
		),

		// Lifecycle hooks
		fx.Invoke(func(
			lc fx.Lifecycle,
			schedulerSvc service.SchedulerService,
			gameUseCase *usecase.GameUseCase,
			wsService service.WebSocketService,
		) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					logger.Info(ctx, "KubeGame backend starting...")

					// Start scheduler service
					if err := schedulerSvc.Start(ctx); err != nil {
						return err
					}

					// Start a sample game for testing
					gameID := "test-game"
					if err := gameUseCase.StartGame(ctx, gameID); err != nil {
						return err
					}

					logger.Info(ctx, "Game %s started successfully", gameID)

					// Start HTTP server for WebSocket connections
					go func() {
						http.Handle("/ws", wsService)
						logger.Info(ctx, "WebSocket server starting on :8080/ws")
						if err := http.ListenAndServe(":8080", nil); err != nil {
							logger.Error(ctx, "HTTP server failed: %v", err)
						}
					}()

					return nil
				},
				OnStop: func(ctx context.Context) error {
					logger.Info(ctx, "KubeGame backend stopping...")
					return schedulerSvc.Stop(ctx)
				},
			})
		}),
	)
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		logger.Error(ctx, "Failed to start KubeGame backend: %v", err)
	}

	// Wait for the application to finish
	sig := app.Wait()
	logger.Info(ctx, "KubeGame backend stopped with signal: %v", sig)

	// Stop the application gracefully
	if err := app.Stop(ctx); err != nil {
		logger.Error(ctx, "Failed to stop KubeGame backend: %v", err)
	}
	logger.Info(ctx, "KubeGame backend stopped successfully")
}
