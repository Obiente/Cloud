package gameservers

import (
	"context"

	gameserversv1 "api/gen/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
)

// SendGameServerCommand sends a command to a running game server
func (s *Service) SendGameServerCommand(ctx context.Context, req *connect.Request[gameserversv1.SendGameServerCommandRequest]) (*connect.Response[gameserversv1.SendGameServerCommandResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	command := req.Msg.GetCommand()

	if command == "" {
		return connect.NewResponse(&gameserversv1.SendGameServerCommandResponse{
			Success:      false,
			ErrorMessage: &[]string{"command cannot be empty"}[0],
		}), nil
	}

	// Check permissions
	if err := s.checkGameServerPermission(ctx, gameServerID, "manage"); err != nil {
		return nil, err
	}

	// Get game server manager
	manager, err := s.getGameServerManager()
	if err != nil {
		return connect.NewResponse(&gameserversv1.SendGameServerCommandResponse{
			Success:      false,
			ErrorMessage: &[]string{err.Error()}[0],
		}), nil
	}

	// Send command to game server
	err = manager.SendGameServerCommand(ctx, gameServerID, command)
	if err != nil {
		return connect.NewResponse(&gameserversv1.SendGameServerCommandResponse{
			Success:      false,
			ErrorMessage: &[]string{err.Error()}[0],
		}), nil
	}

	return connect.NewResponse(&gameserversv1.SendGameServerCommandResponse{
		Success: true,
	}), nil
}

