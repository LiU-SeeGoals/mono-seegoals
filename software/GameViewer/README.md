# Game Viewer

## Speaking with SSL software
In SSL we've got different surrounding software that we want to listen to, currently it's `Game Controller` and `SSL Vision`

The frontend can't subscribe to multicasts, at least that's not how it was implemented when I took this over, so now that's not how we do it (potential improvement!).

Currently the backend (i.e. node.js) subscribes to the multicasts and then resends them unto a websocket as follows.

```
./src/backend/gameControllerProxy.cjs
Bind UDP socket to SSL_GAME_CONTROLLER_PUBLISH_ADDR:SSL_GAME_CONTROLLER_PUBLISH_PORT

Create websocket server to VITE_SSL_GAME_CONTROLLER_WS_ADDR:VITE_SSL_GAME_CONTROLLER_WS_PORT

When UDP socket receives message, send to websocket.
```

Then in the frontend we open a new websocket and subscribe to `VITE_SSL_GAME_CONTROLLER_WS_ADDR:VITE_SSL_GAME_CONTROLLER_WS_PORT` which we created above. This happens in `./src/hooks/useGameController.ts`.

The `./src/backend/sslVisionProxy.cjs` works in the same way.

## Speaking with AI controller
Our own *AI* controller doesn't multicast, it creates a websocket on `VITE_AI_GAME_VIEWER_SOCKET_ADDR:VITE_AI_GAME_VIEWER_SOCKET_PORT` for us, so here we can simply connect to that. We do this in `./src/hooks/useAIController.ts`.
