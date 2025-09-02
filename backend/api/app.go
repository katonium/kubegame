package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	adapterk8s "github.com/katonium/kubegame/backend/adapter/kubernetes"

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
				infraRepo.NewMemorySessionRepository,
				fx.As(new(repository.GameSessionRepository)),
			),
		),

		// Provide Kubernetes cluster manager
		fx.Provide(
			func() adapterk8s.ClusterManager {
				return kubernetes.NewClusterManager()
			},
		),

		// Provide message handler
		fx.Provide(
			func(gi *usecase.GameInteractor) websocket.MessageHandler {
				return websocket.NewMessageHandler(gi)
			},
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
				func(handler websocket.MessageHandler) service.WebSocketService {
					return websocket.NewWebSocketService(handler)
				},
				fx.As(new(service.WebSocketService)),
			),
		),

		// Provide use cases
		fx.Provide(
			func(
				k8sManager adapterk8s.ClusterManager,
				gameRepo repository.GameRepository) *usecase.GameEngineUseCase {
				return usecase.NewGameEngineUseCase(
					k8sManager, gameRepo)
			},
			func(gameEngine *usecase.GameEngineUseCase) *usecase.GameInteractor {
				return usecase.NewGameInteractor(gameEngine)
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
