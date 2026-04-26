(function () {
  const cfg = window.__CONFIG__ || {};
  const apiBase = cfg.apiBase || "/api";
  const apiKey = cfg.apiKey || "";
  const clusterName = cfg.clusterName || "unknown";

  const form = document.getElementById("identity-form");
  const idField = form.elements["id"];
  const errorEl = document.getElementById("form-error");
  const tbody = document.querySelector("#identities tbody");
  const refreshBtn = document.getElementById("refresh");
  const cancelBtn = document.getElementById("cancel-edit");
  const formTitle = document.getElementById("form-title");
  const badge = document.getElementById("cluster-badge");

  badge.textContent = clusterName;

  async function api(method, path, body) {
    const res = await fetch(apiBase + path, {
      method,
      headers: {
        "Content-Type": "application/json",
        "X-API-Key": apiKey
      },
      body: body ? JSON.stringify(body) : null
    });
    if (res.status === 204) return null;
    const text = await res.text();
    const data = text ? JSON.parse(text) : null;
    if (!res.ok) {
      const msg = (data && data.error) || res.statusText;
      throw new Error(msg);
    }
    return data;
  }

  function clearForm() {
    form.reset();
    idField.value = "";
    cancelBtn.hidden = true;
    formTitle.textContent = "New identity";
    errorEl.textContent = "";
  }

  function fillForm(row) {
    for (const key of ["id", "full_name", "phone", "address", "email", "passport_no"]) {
      if (form.elements[key]) form.elements[key].value = row[key] || "";
    }
    cancelBtn.hidden = false;
    formTitle.textContent = "Edit identity";
    errorEl.textContent = "";
  }

  function render(rows) {
    tbody.innerHTML = "";
    if (!rows || rows.length === 0) {
      const tr = document.createElement("tr");
      const td = document.createElement("td");
      td.colSpan = 5;
      td.textContent = "No identities yet.";
      td.style.color = "#6b7280";
      tr.appendChild(td);
      tbody.appendChild(tr);
      return;
    }
    for (const row of rows) {
      const tr = document.createElement("tr");
      tr.innerHTML =
        "<td></td><td></td><td></td><td></td>" +
        "<td><button data-action='edit'>Edit</button> " +
        "<button class='delete' data-action='delete'>Delete</button></td>";
      tr.children[0].textContent = row.full_name || "";
      tr.children[1].textContent = row.email || "";
      tr.children[2].textContent = row.phone || "";
      tr.children[3].textContent = row.passport_no || "";
      tr.dataset.id = row.id;
      tr.dataset.row = JSON.stringify(row);
      tbody.appendChild(tr);
    }
  }

  async function load() {
    try {
      const rows = await api("GET", "/identities");
      render(rows);
    } catch (e) {
      errorEl.textContent = "Failed to load: " + e.message;
    }
  }

  form.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    errorEl.textContent = "";
    const fd = new FormData(form);
    const body = Object.fromEntries(fd.entries());
    const id = body.id;
    delete body.id;
    try {
      if (id) {
        await api("PUT", "/identities/" + encodeURIComponent(id), body);
      } else {
        await api("POST", "/identities", body);
      }
      clearForm();
      await load();
    } catch (e) {
      errorEl.textContent = e.message;
    }
  });

  cancelBtn.addEventListener("click", clearForm);
  refreshBtn.addEventListener("click", load);

  tbody.addEventListener("click", async (ev) => {
    const btn = ev.target.closest("button[data-action]");
    if (!btn) return;
    const tr = btn.closest("tr");
    const id = tr.dataset.id;
    const row = JSON.parse(tr.dataset.row);
    if (btn.dataset.action === "edit") {
      fillForm(row);
      window.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }
    if (btn.dataset.action === "delete") {
      if (!confirm("Delete " + (row.full_name || id) + "?")) return;
      try {
        await api("DELETE", "/identities/" + encodeURIComponent(id));
        await load();
      } catch (e) {
        errorEl.textContent = e.message;
      }
    }
  });

  load();
})();
