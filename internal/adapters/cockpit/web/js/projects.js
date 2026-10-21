// The project list (#/projects): the per-user list of projects, each a
// repository with the runtime directory that setup fills. A card per project
// opens it, sets it up for runs or removes its entry (never its files); a card
// adds a project; the guided setup is the form of `markitect setup` and calls
// the same function. Opening a project that is busy is refused by the cockpit
// and says why. The cards are drawn again only when what they show changes,
// and the add and setup forms are never drawn again by a poll, so focus and
// typed values survive the four-second observation.

import { api } from "./api.js";
import { h, clear, append, announce, pagehead, about, areaIcon } from "./dom.js";
import { store } from "./store.js";
import { refreshState } from "./shell.js";

// ui is what the cards show beyond the list: the entry whose removal asks
// for confirmation and the last message.
const ui = { removing: null, message: "", busy: "" };
// drafts keep a setup form's values while the owner leaves and returns.
const drafts = new Map();

export function renderProjects(content, setupFor) {
  const notices = h("div", { class: "notices", role: "status" });
  const keep = h("div", {});
  const cards = h("div", { class: "projcards" });
  const addCard = addForm();
  const setupSlot = h("div", {});
  const page = h("div", { class: "page" },
    pagehead("Projects", "Open, add and set up the projects the cockpit works on.", {
      about: about("How this works",
        h("p", {}, "A project is a repository with the runtime directory that setup fills."),
        h("p", {}, "Opening a project switches the cockpit to it. That is refused while the open project runs a run you started here, works in a model session or writes an accepted proposal."),
        h("p", {}, "Removing a project forgets its entry only: the repository and the runtime directory stay.")) }),
    notices, keep, cards, setupSlot, addCard.el);
  content.append(page);

  // The address names the project to set up; one that is not in the list
  // shows no form. The form is built once per project, so a poll never
  // replaces what the owner typed.
  let wanted = setupFor;
  let shownSetup = null;
  const showSetup = (view) => {
    const project = wanted ? (view.projects || []).find((p) => p.name === wanted) : null;
    const key = project ? project.name : null;
    if (key === shownSetup) return;
    shownSetup = key;
    clear(setupSlot);
    if (project) setupSlot.append(setupCard(project, view));
  };
  const drawn = {};
  const once = (key, value, fn) => {
    const sig = JSON.stringify(value);
    if (drawn[key] === sig) return;
    drawn[key] = sig;
    fn();
  };
  const draw = (s) => {
    if (!page.isConnected) return unsubscribe();
    const view = s.data.projects;
    if (!view) {
      once("notices", [s.errors.projects?.message], () => clear(notices).append(
        s.errors.projects ? h("p", { class: "refusal", role: "alert" }, `The project list is not readable: ${s.errors.projects.message}`) : h("p", { class: "note" }, "Reading the project list…")));
      return;
    }
    once("notices", [view.available, view.reason, view.error, view.openError, view.warning, view.current && view.busy, view.current?.name, ui.message], () => drawNotices(notices, view));
    once("keep", [view.current, view.available, view.error, ui.busy], () => drawKeep(keep, view));
    once("cards", [view, ui.removing, ui.busy], () => drawCards(cards, view, () => draw(store)));
    once("add", [view.available, view.error], () => addCard.update(view));
    showSetup(view);
  };
  // Cards ask the page to show the setup form of a project, and the form asks
  // to be closed.
  page.addEventListener("setup", (e) => {
    wanted = e.detail;
    history.replaceState(null, "", `#/projects?setup=${encodeURIComponent(wanted)}`);
    draw(store);
    setupSlot.querySelector("h3")?.focus();
  });
  page.addEventListener("closesetup", () => {
    wanted = null;
    history.replaceState(null, "", "#/projects");
    draw(store);
  });
  const unsubscribe = store.subscribe(draw);
  if (store.data.projects) draw(store);
  else refreshState();
}

function drawNotices(el, view) {
  clear(el);
  if (!view.available) el.append(h("p", { class: "refusal", role: "alert" }, view.reason));
  if (view.error) el.append(h("p", { class: "refusal", role: "alert" }, view.error));
  if (view.openError) el.append(h("p", { class: "refusal", role: "alert" }, view.openError));
  if (view.warning) el.append(h("p", { class: "note" }, view.warning));
  if (ui.message) el.append(h("p", { class: "refusal", role: "alert" }, ui.message));
  if (view.current && view.busy) el.append(h("p", { class: "note" }, `${view.current.name} is busy and stays open: ${view.busy}. Opening another project waits for that.`));
  if (view.available && !view.error && !view.projects.length && !view.current) {
    el.append(h("p", { class: "note" }, "No project yet. Add a repository below."));
  }
}

// drawKeep offers to keep the project the flags opened.
function drawKeep(el, view) {
  clear(el);
  const c = view.current;
  if (!c || c.saved || !view.available || view.error) return;
  const name = h("input", { id: "keep-name", type: "text", value: c.name, autocomplete: "off", spellcheck: "false" });
  const button = h("button", { class: "btn act", type: "button", disabled: Boolean(ui.busy) }, "Keep in the project list");
  button.addEventListener("click", () => run("keep", async () => {
    await act(api.keepProject(name.value.trim()));
    announce(`${name.value.trim()} is kept in the project list.`);
  }));
  el.append(h("section", { class: "card keepcard", "aria-labelledby": "keep-title" },
    h("div", { class: "cardhead" }, h("span", { class: "emblem" }, areaIcon("plus")), h("div", { class: "who" },
      h("h3", { id: "keep-title" }, `${c.name} is open from the command line`),
      h("span", { class: "sub digest", title: c.repository }, c.repository))),
    h("p", { class: "note", title: "Explicit state directories are not kept." }, `Keep it to add the repository, its model selection and ${c.runtime ? "its runtime directory" : "no runtime (set it up afterwards)"} to the list.`),
    h("div", { class: "keeprow" },
      h("div", { class: "field" }, h("label", { for: "keep-name" }, "Name in the list"), name),
      button)));
}

function drawCards(el, view, redraw) {
  // Focus stays on the same control across a redraw.
  const focused = el.contains(document.activeElement) ? document.activeElement.dataset.key : null;
  clear(el);
  if (!view.projects.length) return;
  const locked = Boolean(view.busy);
  for (const p of view.projects) {
    const key = (action) => `${action}:${p.name}`;
    const removing = ui.removing === p.name;
    const head = h("div", { class: "cardhead" },
      h("span", { class: "emblem initial", "aria-hidden": "true" }, (p.name || "?").slice(0, 1).toUpperCase()),
      h("div", { class: "who" },
        h("h3", { id: `proj-${p.name}` }, p.name),
        h("span", { class: `state ${p.setUp ? "st-done" : "st-waiting"}` }, h("span", { class: "lamp" }), p.setUp ? "Set up" : "Not set up")),
      p.current ? h("span", { class: "kind current" }, "Open") : null);
    const body = h("dl", { class: "facts" },
      h("dt", {}, "Repository"), h("dd", { class: "digest", title: p.repository }, p.repository),
      h("dt", {}, "Runtime"), h("dd", {}, p.setUp ? h("span", { class: "digest", title: p.runtime }, p.runtime) : h("span", { class: "muted" }, "None: no runs or model sessions until set up")));
    const actions = h("div", { class: "actions" });
    if (p.current) {
      actions.append(h("a", { class: "btn act small", href: "#/", "data-key": key("go") }, "Go to the project"));
    } else {
      actions.append(button("Open", { key: key("open"), cls: "btn act small", disabled: locked || ui.busy, title: locked ? `The open project is busy: ${view.busy}` : "", onclick: () => run("open", async () => {
        await act(api.openProject(p.name), true);
        store.reset();
        await refreshState();
        await refreshState();
        announce(`${p.name} is open.`);
        location.hash = "#/";
      }) }));
    }
    actions.append(h("a", { class: "btn small", href: `#/projects?setup=${encodeURIComponent(p.name)}`, "data-key": key("setup"), onclick: (e) => {
      e.preventDefault();
      el.closest(".page").dispatchEvent(new CustomEvent("setup", { detail: p.name }));
    } }, p.setUp ? "Set up again" : "Set up runs"));
    if (!removing) {
      actions.append(button("Remove…", { key: key("remove"), cls: "btn quiet small end", disabled: ui.busy, onclick: () => { ui.removing = p.name; ui.message = ""; redrawAll(redraw, key("confirm")); } }));
    }
    const confirm = [];
    if (removing) {
      confirm.push(h("div", { class: "confirm", role: "group", "aria-label": `Remove ${p.name} from the list` },
        h("p", { class: "note", role: "status" }, "Removes the entry from the list. The repository and the runtime directory are not touched."),
        p.current && view.busy ? h("p", { class: "refusal", role: "alert" }, `It is open and busy: ${view.busy}.`) : null,
        h("div", { class: "actions" },
          button("Remove from the list", { key: key("confirm"), cls: "btn act small", disabled: ui.busy, onclick: () => run("remove", async () => {
            ui.removing = null;
            await act(api.removeProject(p.name));
            announce(`${p.name} is removed from the list.`);
          }) }),
          button("Keep it", { key: key("cancel"), cls: "btn small", disabled: ui.busy, onclick: () => { ui.removing = null; redrawAll(redraw, key("remove")); } }))));
    }
    el.append(h("section", { class: `card projcard${p.current ? " is-current" : ""}`, "aria-labelledby": `proj-${p.name}` }, head, body, actions, confirm));
  }
  if (focused) el.querySelector(`[data-key="${CSS.escape(focused)}"]`)?.focus();
}

function button(text, { key, cls, disabled, title, onclick }) {
  return h("button", { class: cls, type: "button", "data-key": key, disabled: Boolean(disabled), title: title || null, onclick }, text);
}

// redrawAll draws the cards again and puts the focus on the named control.
function redrawAll(redraw, key) {
  redraw();
  document.querySelector(`.projcards [data-key="${CSS.escape(key)}"]`)?.focus();
}

// run performs one action of the page with its busy state and message.
async function run(what, fn) {
  ui.busy = what;
  ui.message = "";
  store.notify();
  try {
    await fn();
  } catch (err) {
    ui.message = err.message || String(err);
  }
  ui.busy = "";
  store.notify();
}

// act sends a change and takes the answer as the new project list. A project
// switch also resets what was read from the earlier project.
async function act(request) {
  const view = await request;
  store.data.projects = view;
  delete store.errors.projects;
  return view;
}

// ---------- add ----------

function addForm() {
  const f = {
    name: field("add-name", "Name", "Default: the directory name"),
    repository: field("add-repository", "Repository", "Absolute path of the root"),
    runtime: field("add-runtime", "Runtime directory (optional)", "Filled by markitect setup"),
    model: field("add-model", "Model manifest", "Relative to the repository, or absolute", ".markitect/model.yaml"),
  };
  const deps = h("textarea", { id: "add-deps", rows: "2", spellcheck: "false", autocomplete: "off" });
  const status = h("p", { class: "note", role: "status", tabindex: "-1" });
  const submit = h("button", { class: "btn act", type: "submit" }, "Add the project");
  const form = h("form", { class: "settings addform", novalidate: true },
    h("div", { class: "grid4" }, f.repository.el, f.name.el, f.runtime.el),
    h("details", {}, h("summary", {}, "Model selection"),
      h("div", { class: "grid4" }, f.model.el, h("div", { class: "field" }, h("label", { for: "add-deps" }, "Dependency models"), deps, h("span", { class: "faint" }, "One path per line, as for --dependency")))),
    h("div", { class: "actions" }, submit), status);
  const out = { el: null, update: () => {} };
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    submit.disabled = true;
    status.textContent = "Adding…";
    try {
      const view = await api.addProject({
        name: f.name.input.value.trim(), repository: f.repository.input.value.trim(), runtime: f.runtime.input.value.trim(),
        model: f.model.input.value.trim(), dependencies: deps.value.split("\n").map((d) => d.trim()).filter(Boolean),
      });
      store.data.projects = view;
      delete store.errors.projects;
      store.notify();
      for (const x of [f.name, f.repository, f.runtime]) x.input.value = "";
      deps.value = "";
      status.textContent = "The project is in the list.";
      announce("The project is added to the list.");
    } catch (err) {
      status.textContent = `Not added: ${err.message || err}`;
      status.focus();
    }
    submit.disabled = false;
  });
  out.el = h("section", { class: "card addcard", "aria-labelledby": "add-title" },
    h("div", { class: "cardhead" }, h("span", { class: "emblem" }, areaIcon("plus")), h("div", { class: "who" },
      h("h3", { id: "add-title" }, "Add a project"),
      h("span", { class: "sub" }, "A repository with a model. The runtime directory is optional."))),
    form);
  out.update = (view) => {
    const off = !view || !view.available || Boolean(view.error);
    for (const input of form.querySelectorAll("input, textarea, button")) input.disabled = off;
  };
  return out;
}

function field(id, label, hint, value = "") {
  const input = h("input", { id, type: "text", value, autocomplete: "off", spellcheck: "false" });
  return { input, el: h("div", { class: "field" }, h("label", { for: id }, label), input, hint ? h("span", { class: "faint" }, hint) : null) };
}

// ---------- guided setup ----------

// setupCard is the form of markitect setup for a project. Writing takes two
// steps: the review names the files and the directory.
function setupCard(project, view) {
  const name = project.name;
  const sep = (view.runtimeBase || "").includes("\\") ? "\\" : "/";
  const draft = drafts.get(name) || {
    runtimeDir: view.runtimeBase ? `${view.runtimeBase}${sep}${name}` : "",
    image: "", credential: "", agentModel: "", effort: view.setup?.effort || "", engine: view.setup?.engine || "",
    checks: [{ profile: "", executable: "" }], store: "", deadline: "",
  };
  drafts.set(name, draft);
  const text = (id, label, key, hint) => {
    const input = h("input", { id: `setup-${id}`, type: "text", value: draft[key], autocomplete: "off", spellcheck: "false" });
    input.addEventListener("input", () => { draft[key] = input.value; });
    return h("div", { class: "field" }, h("label", { for: `setup-${id}` }, label), input, hint ? h("span", { class: "faint" }, hint) : null);
  };
  const checks = h("div", { class: "checkrows" });
  const drawChecks = () => {
    clear(checks);
    draft.checks.forEach((c, i) => {
      const profile = h("input", { type: "text", value: c.profile, "aria-label": `Check profile ${i + 1}`, autocomplete: "off", spellcheck: "false" });
      const exe = h("input", { type: "text", value: c.executable, "aria-label": `Executable of check profile ${i + 1}`, autocomplete: "off", spellcheck: "false" });
      profile.addEventListener("input", () => { c.profile = profile.value; });
      exe.addEventListener("input", () => { c.executable = exe.value; });
      const remove = h("button", { class: "btn quiet small", type: "button", onclick: () => {
        draft.checks.splice(i, 1);
        if (!draft.checks.length) draft.checks.push({ profile: "", executable: "" });
        drawChecks();
        checks.querySelector("input")?.focus();
      } }, "Remove");
      checks.append(h("div", { class: "checkrow" }, profile, h("span", { "aria-hidden": "true" }, "="), exe, remove));
    });
  };
  drawChecks();
  const form = h("form", { class: "settings", novalidate: true },
    h("div", { class: "grid4" },
      text("runtime", "Runtime directory", "runtimeDir", "Absolute, outside the repository; setup writes runtime.yaml, run.json and state/ there"),
      text("image", "Workspace image", "image", "Image reference the Agent containers run"),
      text("credential", "Credential file", "credential", "Absolute path of the private token file; its bytes are never sent here"),
      text("model", "Agent model", "agentModel", "The model of every Agent profile"),
      text("effort", "Effort", "effort", ""),
      text("engine", "Container engine", "engine", "Absolute path of the executable")),
    h("fieldset", { class: "plate" }, h("legend", {}, "Check profiles"),
      h("p", { class: "note" }, "One PROFILE = EXECUTABLE per check profile the project selects; setup names a missing one."),
      checks,
      h("div", { class: "addrow" }, h("button", { class: "btn small", type: "button", onclick: () => { draft.checks.push({ profile: "", executable: "" }); drawChecks(); checks.querySelector(".checkrow:last-child input")?.focus(); } }, "Add a check"))),
    h("div", { class: "grid4" },
      text("store", "Store label (optional)", "store", "Default: the model id"),
      text("deadline", "Deadline (optional)", "deadline", "Absolute RFC 3339 UTC time, such as 2026-11-01T18:00:00Z")));
  const result = h("div", { class: "setupresult", role: "status", tabindex: "-1" });
  const review = h("div", {});
  const actions = h("div", { class: "actions" });
  const state = { step: "form", busy: false };
  const body = () => ({
    runtimeDir: draft.runtimeDir.trim(), image: draft.image.trim(), credential: draft.credential.trim(), agentModel: draft.agentModel.trim(),
    effort: draft.effort.trim(), engine: draft.engine.trim(), store: draft.store.trim(), deadline: draft.deadline.trim(),
    checks: draft.checks.map((c) => ({ profile: c.profile.trim(), executable: c.executable.trim() })).filter((c) => c.profile || c.executable),
  });
  const draw = () => {
    clear(actions);
    clear(review);
    if (state.step === "form") {
      form.hidden = false;
      actions.append(h("button", { class: "btn act", type: "button", onclick: toReview }, "Review…"),
        h("button", { class: "btn", type: "button", onclick: () => form.closest(".page").dispatchEvent(new CustomEvent("closesetup")) }, "Close"));
      return;
    }
    const b = body();
    form.hidden = true;
    append(review, [
      h("div", { class: "confirm", role: "group", "aria-label": "Confirm the setup" },
        h("h4", {}, `Writes runtime.yaml and run.json in ${b.runtimeDir}`),
        h("p", { class: "note" }, "Setup observes the engine and the image, creates the directory with its state directory and writes the two files. It starts no container and reads no credential."),
        h("dl", { class: "facts" },
          h("dt", {}, "Project"), h("dd", {}, `${name} (${project.repository})`),
          h("dt", {}, "Image"), h("dd", { class: "digest" }, b.image),
          h("dt", {}, "Credential file"), h("dd", { class: "digest" }, b.credential),
          h("dt", {}, "Agent model"), h("dd", {}, `${b.agentModel}${b.effort ? `, effort ${b.effort}` : ""}`),
          h("dt", {}, "Container engine"), h("dd", { class: "digest" }, b.engine),
          h("dt", {}, "Checks"), h("dd", {}, b.checks.length ? b.checks.map((c) => h("div", { class: "digest" }, `${c.profile}=${c.executable}`)) : "none"),
          b.store ? [h("dt", {}, "Store label"), h("dd", {}, b.store)] : null,
          b.deadline ? [h("dt", {}, "Deadline"), h("dd", {}, b.deadline)] : null))]);
    actions.append(
      h("button", { class: "btn act", type: "button", "data-primary": "", disabled: state.busy, onclick: write }, state.busy ? "Setting up…" : "Write the files"),
      h("button", { class: "btn", type: "button", disabled: state.busy, onclick: () => { state.step = "form"; draw(); form.querySelector("input")?.focus(); } }, "Back to the form"));
    actions.querySelector("[data-primary]")?.focus();
  };
  const toReview = () => {
    clear(result);
    const b = body();
    const missing = [["runtime directory", b.runtimeDir], ["workspace image", b.image], ["credential file", b.credential], ["Agent model", b.agentModel]].filter(([, v]) => !v).map(([n]) => n);
    if (missing.length) {
      result.append(h("p", { class: "refusal", role: "alert" }, `Setup needs the ${missing.join(", the ")}.`));
      result.focus();
      return;
    }
    state.step = "review";
    draw();
  };
  const write = async () => {
    state.busy = true;
    clear(result);
    draw();
    try {
      const view = await api.setupProject(name, body());
      store.data.projects = view;
      delete store.errors.projects;
      const r = view.setupResult;
      if (r && r.status === "passed") {
        drafts.delete(name);
        if (view.opened) {
          store.reset();
          await refreshState();
          await refreshState();
        } else {
          store.notify();
        }
        result.append(setupSuccess(r, view));
        announce("Setup wrote the runtime files.");
        form.hidden = true;
        state.step = "done";
        clear(actions);
        clear(review);
        append(actions, [view.opened ? h("a", { class: "btn act", href: "#/runs" }, "Go to Runs") : null, h("a", { class: "btn", href: "#/projects" }, "Back to the projects")]);
      } else {
        // Setup's own refusal, word for word.
        result.append(h("div", { class: "refusal", role: "alert" },
          h("strong", {}, r?.status === "failed" ? "Setup refused" : "Setup did not complete"),
          h("pre", { class: "refusaltext" }, r?.error || "Setup reported no reason.")));
        state.step = "form";
      }
    } catch (err) {
      result.append(h("p", { class: "refusal", role: "alert" }, `Not set up: ${err.message || err}`));
      state.step = "form";
    }
    state.busy = false;
    if (state.step !== "done") draw();
    result.focus();
  };
  draw();
  const el = h("section", { class: "card setupcard", "aria-labelledby": "setup-title" },
    h("div", { class: "cardhead" }, h("span", { class: "emblem initial", "aria-hidden": "true" }, (name || "?").slice(0, 1).toUpperCase()), h("div", { class: "who" },
      h("h3", { id: "setup-title", tabindex: "-1" }, `Set up runs for ${name}`),
      h("span", { class: "sub" }, "Prepares the runtime directory for runs and model sessions."))),
    h("div", { class: "settings" },
      about("What this does", h("p", {}, "It calls the function behind markitect setup with the project's repository and model selection and writes exactly what it writes. Its refusals are shown as they are.")),
      form, review, result, actions));
  return el;
}

function setupSuccess(r, view) {
  return h("div", { class: "written" },
    h("p", {}, h("span", { class: "outcome st-done" }, h("span", { class: "lamp" }), "Setup passed"), view.opened ? " The project is open with runs enabled." : null),
    view.warning ? h("p", { class: "refusal" }, view.warning) : null,
    r.bound ? h("dl", { class: "facts" },
      h("dt", {}, "Image"), h("dd", { class: "digest", title: r.bound.imageID }, r.bound.imageID),
      h("dt", {}, "Agent daemon"), h("dd", {}, r.bound.agentdVersion),
      h("dt", {}, "CLI"), h("dd", {}, r.bound.cliVersion),
      h("dt", {}, "Engine"), h("dd", {}, r.bound.engineVersion)) : null,
    h("ul", { class: "rows" }, (r.files || []).map((f) => h("li", {}, h("span", { class: "row" }, h("span", { class: "title digest", title: f.path }, f.path), h("span", { class: "meta" }, f.action))))));
}
