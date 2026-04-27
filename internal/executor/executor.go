package executor

import (
	"context"
	"encoding/json"
	"io"
	"iter"
	"log/slog"
	"sync"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/aleksclark/crush-a2a/internal/bridge"
	"github.com/aleksclark/crush-a2a/internal/crush"
)

var _ a2asrv.AgentExecutor = (*CrushExecutor)(nil)

// CrushExecutor implements a2asrv.AgentExecutor by bridging to the Crush native API.
type CrushExecutor struct {
	Crush         *crush.Client
	WorkspacePath string
	Logger        *slog.Logger

	mu          sync.Mutex
	workspaceID string
}

func (e *CrushExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		prompt := bridge.ExtractPromptText(execCtx.Message.Parts)
		if prompt == "" {
			yield(nil, a2a.ErrInvalidParams)
			return
		}

		wsID, err := e.ensureWorkspace(ctx)
		if err != nil {
			e.Logger.Error("failed to ensure workspace", "error", err)
			yield(nil, err)
			return
		}

		sess, err := e.Crush.CreateSession(ctx, wsID, "A2A task "+string(execCtx.TaskID))
		if err != nil {
			e.Logger.Error("failed to create session", "error", err)
			yield(nil, err)
			return
		}

		e.Logger.Info("Execute: sending prompt",
			"task_id", execCtx.TaskID,
			"context_id", execCtx.ContextID,
			"workspace_id", wsID,
			"session_id", sess.ID,
		)

		sseStream, err := e.Crush.SubscribeEvents(ctx, wsID)
		if err != nil {
			e.Logger.Error("failed to subscribe to events", "error", err)
			yield(nil, err)
			return
		}
		defer sseStream.Close()

		err = e.Crush.SendMessage(ctx, wsID, crush.AgentMessage{
			SessionID: sess.ID,
			Prompt:    prompt,
		})
		if err != nil {
			e.Logger.Error("failed to send message", "error", err)
			yield(nil, err)
			return
		}

		if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil) {
			return
		}

		finished := make(chan struct{})
		var finalState a2a.TaskState
		var finishMsg string

		go func() {
			defer close(finished)
			crush.ReadSSE(ctx, sseStream, func(payload crush.SSEPayload) error {
				if payload.Type != crush.PayloadTypeMessage {
					return nil
				}
				var ev crush.SSEEvent[crush.Message]
				if err := json.Unmarshal(payload.Payload, &ev); err != nil {
					return nil
				}
				m := ev.Payload
				if m.SessionID != sess.ID || m.Role != crush.RoleAssistant {
					return nil
				}
				if finish := m.FinishPart(); finish != nil {
					finalState = bridge.CrushFinishToA2AState(finish.Reason)
					finishMsg = finish.Message
					return io.EOF
				}
				return nil
			})
		}()

		select {
		case <-finished:
		case <-time.After(5 * time.Minute):
			finalState = a2a.TaskStateFailed
			finishMsg = "timeout waiting for agent response"
		case <-ctx.Done():
			finalState = a2a.TaskStateCanceled
		}

		if finalState == "" {
			finalState = a2a.TaskStateCompleted
		}

		messages, err := e.Crush.GetMessages(ctx, wsID, sess.ID)
		if err != nil {
			e.Logger.Error("failed to get messages", "error", err)
			if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateFailed,
				a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart("failed to retrieve messages"))), nil) {
				return
			}
			return
		}

		for _, artifact := range bridge.CrushMessagesToA2AArtifacts(messages) {
			if !yield(a2a.NewArtifactEvent(execCtx, artifact.Parts...), nil) {
				return
			}
		}

		var statusMsg *a2a.Message
		if finishMsg != "" {
			statusMsg = a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(finishMsg))
		}

		yield(a2a.NewStatusUpdateEvent(execCtx, finalState, statusMsg), nil)
	}
}

func (e *CrushExecutor) Cancel(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}

func (e *CrushExecutor) ensureWorkspace(ctx context.Context) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.workspaceID != "" {
		return e.workspaceID, nil
	}

	ws, err := e.Crush.CreateWorkspace(ctx, e.WorkspacePath)
	if err != nil {
		return "", err
	}

	e.Logger.Info("workspace created", "id", ws.ID, "path", ws.Path)

	if err := e.Crush.SkipPermissions(ctx, ws.ID); err != nil {
		e.Logger.Warn("failed to set skip permissions", "error", err)
	}

	if err := e.Crush.InitAgent(ctx, ws.ID); err != nil {
		e.Logger.Warn("failed to init agent", "error", err)
	}

	e.workspaceID = ws.ID
	return ws.ID, nil
}
