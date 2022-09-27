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
    successAlert(data.message);
  })

  publicChannel.bind("app-stopped", (data) => {
    warningAlert(data.message);
  })

  publicChannel.bind("next-run-event", (data) => {
  })

  publicChannel.bind("schedule-changed-event", (data) => {
  })

  publicChannel.bind("host-service-status-change", (data) => {
    attention.toast({
      msg: data.message,
      icon: "info",
      timer: 10 * 1000,
      showCloseButton: true
    });

    //   remove table row if it exist
    const tr = document.getElementById(`host-service-tr-${data.host_service_id}`)
    if (tr) {
      tr.parentNode.removeChild(tr);
    }

    //  update tables if they exist
    const table = document.getElementById(`${data.status}-table`);
    if (table) {
      const trHtml = `
      <td>
        <i class="${data.icon}"></i>
        ${data.service_name}
        <span class="badge bg-secondary clicker-badge" onclick="checkNow(${data.host_service_id}, '${data.status}')">
          Check Now
        </span>
      </td>
      <td>
        ${data.last_check}
      </td>
      <td>${data.service_name}</td>
      `;
      const tr = table.tBodies[0].insertRow(-1);
      tr.setAttribute('id', `host-service-tr-${data.host_service_id}`);
      tr.innerHTML = trHtml;
    }
  })

  publicChannel.bind("hs-count-changed", (data) => {
    const { healthy_count, pending_count, problem_count, warning_count } = data;

    // Update Overview Page
    const healthySpan = document.getElementById("healthy-count");
    if (healthySpan) {
      healthySpan.innerHTML = healthy_count;
    }
    const warningSpan = document.getElementById("warning_count");
    if (warningSpan) {
      warningSpan.innerHTML = warning_count;
    }
    const problemSpan = document.getElementById("problem-count");
    if (problemSpan) {
      problemSpan.innerHTML = problem_count;
    }
    const pendingSpan = document.getElementById("pending_count");
    if (pendingSpan) {
      pendingSpan.innerHTML = pending_count
    }
  })


  /*Also listen for events:
  * - service goes up
  * - service goes down
  * - service status changed
  * - monitoring is turned off
  * */
}