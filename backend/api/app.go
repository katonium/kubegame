package main

import (
	"context"
	"net/http"
	"fmt"
	"os"

	"github.com/katonium/kubegame/backend/domain/repository"
	"github.com/katonium/kubegame/backend/domain/service"
	"github.com/katonium/kubegame/backend/infrastructure/kubernetes"
	infraRepo "github.com/katonium/kubegame/backend/infrastructure/repository"
	infraService "github.com/katonium/kubegame/backend/infrastructure/service"
	"github.com/katonium/kubegame/backend/infrastructure/websocket"
	"github.com/katonium/kubegame/backend/usecase"
	"github.com/katonium/kubegame/backend/util/logger"
	"go.uber.org/fx"
	"k8s.io/client-go/kubernetes/fake"
)

var defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

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
			fx.Annotate(
				infraRepo.NewMemorySessionRepository,
				fx.As(new(repository.GameSessionRepository)),
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
				func(sessionRepo repository.GameSessionRepository,
					gameRepo repository.GameRepository,
					podRepo repository.PodRepository,
					nodeRepo repository.NodeRepository,
					kubernetesService service.KubernetesService) service.GameSessionService {
					return infraService.NewGameSessionService(sessionRepo, gameRepo, podRepo, nodeRepo, kubernetesService)
				},
				fx.As(new(service.GameSessionService)),
			),
			fx.Annotate(
				func(gameSessionService service.GameSessionService,
					gameEngine *usecase.GameEngineUseCase,
					gameUseCase *usecase.GameUseCase) service.WebSocketService {
					return websocket.NewWebSocketService(gameSessionService, gameEngine, gameUseCase)
				},
				fx.As(new(service.WebSocketService)),
			),
		),

		// Provide use cases
		fx.Provide(
			func(gameRepo repository.GameRepository,
				podRepo repository.PodRepository,
				nodeRepo repository.NodeRepository,
				sessionRepo repository.GameSessionRepository,
				k8sService service.KubernetesService) *usecase.GameEngineUseCase {
				return usecase.NewGameEngineUseCase(gameRepo, podRepo, nodeRepo, sessionRepo, nil, k8sService)
			},
			func(gameRepo repository.GameRepository,
				podRepo repository.PodRepository,
				nodeRepo repository.NodeRepository,
				sessionRepo repository.GameSessionRepository,
				k8sService service.KubernetesService,
				schedulerSvc service.SchedulerService,
				gameEngine *usecase.GameEngineUseCase) *usecase.GameUseCase {
				return usecase.NewGameUseCase(gameRepo, podRepo, nodeRepo, sessionRepo, k8sService, schedulerSvc, nil, gameEngine)
			},
		),

		// Lifecycle hooks
		fx.Invoke(func(
			lc fx.Lifecycle,
			schedulerSvc service.SchedulerService,
			wsService service.WebSocketService,
		) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					logger.Info(ctx, "KubeGame backend starting...")

					// Start scheduler service
					if err := schedulerSvc.Start(ctx); err != nil {
						return err
					}

					logger.Info(ctx, "KubeGame backend ready for connections")

					// Start HTTP server for WebSocket connections
					go func() {
						http.Handle("/ws", wsService)
						logger.Info(ctx, "WebSocket server starting on :%s/ws", port)
						if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
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
	
	// Run the application and block until shutdown
	app.Run()
}
