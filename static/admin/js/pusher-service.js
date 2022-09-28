const tableNames = ['healthy', 'warning', 'problem', 'pending'];

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

  publicChannel.bind("schedule-changed-event", (data) => {
    const scheduleTable = document.getElementById('schedule-table');
    if (scheduleTable) {
      const {
        host_service_id,
        next_run,
        last_check,
        schedule,
        service_name,
        host_name
      } = data;

      const emptyRow = document.getElementById("no-rows");
      if (emptyRow) {
        emptyRow.parentNode.removeChild(emptyRow);
      }

      const innerHTML = `
        <td>${host_name}</td>
        <td>${service_name}</td>
        <td>${schedule}</td>
        <td>${last_check}</td>
        <td>${next_run}</td>
      `;

      const existRow = document.getElementById(`schedule-${host_service_id}`)
      if (existRow) {
        existRow.innerHTML = innerHTML;
      } else {
        const tr = scheduleTable.tBodies[0].insertRow(-1);
        tr.setAttribute('id', `schedule-${host_service_id}`)
        tr.innerHTML = innerHTML;
      }
    }
  })

  publicChannel.bind("schedule-removed-event", (data) => {
    const scheduleTable = document.getElementById('schedule-table');
    if (scheduleTable) {
      const {
        host_service_id,
      } = data;

      const existRow = document.getElementById(`schedule-${host_service_id}`)
      if (existRow) {
        existRow.parentNode.removeChild(existRow);
      }

      if (scheduleTable && scheduleTable.rows.length === 1) {
        const rw = scheduleTable.tBodies[0].insertRow(-1)
        rw.setAttribute("id", "no-rows");
        rw.innerHTML = `<td colspan="5">No Schedules</td>`;
      }
    }
  })

  publicChannel.bind("app-started", (data) => {
    successAlert(data.message);
    const toggle = document.getElementById('monitoring-live');
    if (toggle) {
      toggle.checked = true;
    }
  })

  publicChannel.bind("app-stopped", (data) => {
    warningAlert(data.message);
    const toggle = document.getElementById('monitoring-live');
    if (toggle) {
      toggle.checked = false;
    }

    const scheduleTable = document.getElementById('schedule-table');
    if (scheduleTable) {
      scheduleTable.tBodies[0].rows.forEach(tr => {
        tr.parentNode.removeChild(tr);
      })
      const rw = scheduleTable.tBodies[0].insertRow(-1)
      rw.setAttribute("id", "no-rows");
      rw.innerHTML = `<td colspan="5">No Schedules</td>`;
    }
  })

  publicChannel.bind("next-run-event", (data) => {
  })

  publicChannel.bind("host-service-status-change", (data) => {
    attention.toast({
      msg: data.message,
      icon: "info",
      timer: 10 * 1000,
      showCloseButton: true
    });

    // Update Host Page
    removeHostTableTr(data.host_service_id);

    //  update tables if they exist
    addHostTableRow(data);
  //  Update Status pages
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

  function removeHostTableTr(rowId) {
    const tr = document.getElementById(`host-service-tr-${rowId}`)
    if (tr) {
      tr.parentNode.removeChild(tr);

      tableNames.forEach(tableName => {
        const table = document.getElementById(`${tableName}-only-table`);
        if (table && table.rows.length === 1) {
          const rw = table.tBodies[0].insertRow(-1);
          rw.setAttribute('id', 'no-rows');
          rw.innerHTML = `<td colspan="4">No Services</td>`;
        }
      })

    }
  }

  function addHostTableRow(data) {
    const { host_service_id, host_id, host_name, service_name, icon, status, last_check } = data;
    const hostTable = document.getElementById(`${data.status}-table`);
    if (hostTable) {
      const trHtml = `
      <td>
        <i class="${icon}"></i>
        ${service_name}
        <span class="badge bg-secondary clicker-badge" onclick="checkNow(${host_service_id}, '${status}')">
          Check Now
        </span>
      </td>
      <td>
        ${last_check}
      </td>
      <td>${service_name}</td>
      `;
      const tr = hostTable.tBodies[0].insertRow(-1);
      tr.setAttribute('id', `host-service-tr-${host_service_id}`);
      tr.innerHTML = trHtml;
    }

    tableNames.forEach(tableName => {
      const tbl = document.getElementById(`${tableName}-only-table`);
      if (tbl && tableName === status) {
        const emptyRow = document.getElementById(`no-rows`)
        if (emptyRow) {
          emptyRow.parentNode.removeChild(emptyRow);
        }
        const trHtml = `
          <td><a href="/admin/host/${host_id}#${tableName}-content">${host_name}</a></td>
          <td>${service_name}</td>
          <td><span class="badge bg-success">${status}</span></td>
          <td></td>
        `;
        const tr = tbl.tBodies[0].insertRow(-1);
        tr.setAttribute('id', `host-service-tr-${host_service_id}`);
        tr.innerHTML = trHtml;
      }
    })
  }

  /*Also listen for events:
  * - service goes up
  * - service goes down
  * - service status changed
  * - monitoring is turned off
  * */
}