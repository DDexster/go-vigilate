function initPusher(pusherKey) {
  const pusher = new Pusher(pusherKey, {
    authEndPoint: '/pusher/auth',
    wsHost: 'localhost',
    wsPort: 4001,
    forceTLS: false,
    enabledTransports: ['ws', 'wss'],
    disabledTransports: []
  })

  const publicChannel = pusher.subscribe('public-channel');

  publicChannel.bind("test-event", (data) => {
    successAlert(data.message);
  })

  publicChannel.bind("app-started", (data) => {
  })

  publicChannel.bind("next-run-event", (data) => {
  })

  publicChannel.bind("schedule-changed-event", (data) => {
  })
}