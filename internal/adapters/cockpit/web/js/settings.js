// The execution settings form. Its result is a proposal, never a write: the
// cockpit makes the changed .markitect/execution.yaml from the committed
// source, the structural check judges it, and the owner decides on the
// proposal's own page. The draft lives outside the page, so leaving and
// returning keeps it while the committed source is unchanged.

import { api } from "./api.js";
import { h, clear, announce } from "./dom.js";

let draft = null; // { head, settings }

export function renderSettings(app) {
  document.title = "Execution settings – Markitect cockpit";
  clear(app);
  const main = h("main", { id: "main", class: "runs authoring" });
  app.append(main);
  main.append(top("Execution settings"), h("p", { class: "lead", role: "status" }, "Capturing the committed source and preparing it as markitect check does. On a large model this takes a while…"));
  api.settings().then((settings) => {
    if (!main.isConnected) return;
    if (!draft || draft.head !== settings.head) draft = { head: settings.head, settings: structuredClone(settings) };
    clear(main).append(top("Execution settings"), settingsForm(settings, draft.settings));
  }, (err) => {
    if (!main.isConnected) return;
    clear(main).append(top("Execution settings"), h("p", { role: "alert", class: "refusal" }, refusal(err)));
  });
}

export function top(title) {
  return h("div", { class: "pagetop" }, h("a", { class: "home", href: "#/" }, "Markitect cockpit"), h("h1", {}, title));
}

// refusal explains a refused read or proposal in the owner's terms.
export function refusal(err) {
  const text = err.message || String(err);
  if (/uncommitted|not clean|dirty/i.test(text)) {
    return `The working tree differs from its commit. A proposal starts from the committed source, so commit or set aside the open changes first. (${text})`;
  }
  if (/without a repository|--repo/.test(text)) return "Proposals need the cockpit to be started in a repository: markitect cockpit --repo PATH.";
  return text;
}

function settingsForm(committed, s) {
  const form = h("form", { class: "settings", novalidate: true });
  const status = h("p", { class: "note", role: "status", tabindex: -1 });
  const submit = h("button", { class: "btn act", type: "submit" }, "Propose the change");
  form.append(
    h("p", { class: "lead" }, "Changing a value here makes a proposal. The structural check judges the changed file, you review its difference, and only your acceptance writes it to the working tree. Nothing is committed and no Apply starts."),
    h("p", { class: "note" }, `From the committed source at `, h("span", { class: "digest" }, committed.head.slice(0, 12)), `, file .markitect/execution.yaml. Ownership, the top-level and per-Package instructions, scope and checks are not edited here; they stay as written.`),
    budget(s.budget),
    profiles(s.profiles),
    overrides(s),
    team(s, committed),
    h("div", { class: "actions" }, submit, h("button", { class: "btn", type: "button", onclick: () => reset(form, committed) }, "Reset to the committed settings")),
    status);
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    submit.disabled = true;
    status.textContent = "Making the proposal and running the structural check on it…";
    try {
      const proposal = await api.proposeSettings(s);
      draft = null;
      announce(`Proposal ${proposal.id} made.`);
      location.hash = `#/proposal/${encodeURIComponent(proposal.id)}`;
    } catch (err) {
      submit.disabled = false;
      status.textContent = `No proposal made: ${refusal(err)}`;
      status.focus();
    }
  });
  return form;
}

function reset(form, committed) {
  draft = { head: committed.head, settings: structuredClone(committed) };
  form.replaceWith(settingsForm(committed, draft.settings));
}

// field is one labelled input bound to obj[key]. Numbers stay numbers; an
// invalid entry is passed on as it is, so the check names it.
function field(labelText, obj, key, { type = "text", hint, placeholder, id } = {}) {
  id = id || `f-${Math.random().toString(36).slice(2, 9)}`;
  const input = h("input", { id, type, value: obj[key] ?? "", placeholder, inputmode: type === "number" ? "numeric" : undefined, min: type === "number" ? 0 : undefined, "aria-describedby": hint ? `${id}-hint` : undefined });
  input.addEventListener("input", () => {
    obj[key] = type === "number" ? (input.value === "" ? 0 : Number(input.value)) : input.value;
  });
  return h("div", { class: "field" }, h("label", { for: id }, labelText), input, hint && h("span", { class: "faint", id: `${id}-hint` }, hint));
}

function budget(b) {
  return h("fieldset", { class: "plate" }, h("legend", {}, "Budget of one Apply"),
    h("div", { class: "grid4" },
      field("Run time", b, "executionTime", { hint: "such as 25m or 1h30m" }),
      field("Time per Agent turn", b, "turnTimeout", { hint: "such as 10m" }),
      field("Agent turns", b, "agentTurns", { type: "number" }),
      field("Passes", b, "passes", { type: "number" })));
}

function profiles(p) {
  return h("fieldset", { class: "plate" }, h("legend", {}, "Runtime profiles of the roles"),
    h("div", { class: "grid4" },
      field("Execute", p, "execute", { hint: "Managers and their helpers" }),
      field("Verify", p, "verify", { hint: "independent reviewers" })),
    h("p", { class: "note" }, "A name refers to an Agent profile of the Runtime profile given to markitect apply. The structural check cannot see that profile; an unknown name is refused when an Apply starts."));
}

// overrides lists the Packages with their own Execute or Verify profile and
// lets the owner give another local Package its own; an empty input keeps
// the role's profile above.
function overrides(s) {
  s.packages = s.packages || [];
  s.overrides = (s.overrides || []).filter((o) => o.package);
  const box = h("fieldset", { class: "plate" }, h("legend", {}, "Profiles per Package"));
  const body = h("div", {});
  const draw = (focus) => {
    clear(body);
    if (s.overrides.length) {
      const rows = s.overrides.map((o, i) => {
        const input = (key, role) => {
          const el = h("input", { id: `ov-${i}-${key}`, value: o[key], placeholder: `uses ${s.profiles[key] || "the default"}`, "aria-label": `${role} profile of ${o.package}` });
          el.addEventListener("input", () => { o[key] = el.value; });
          return el;
        };
        return h("tr", {}, h("th", { scope: "row" }, o.package), h("td", {}, input("execute", "Execute")), h("td", {}, input("verify", "Verify")),
          h("td", {}, h("button", { class: "btn", type: "button", "aria-label": `Remove the profiles of ${o.package}`, onclick: () => {
            s.overrides.splice(i, 1);
            draw("select");
          } }, "Remove")));
      });
      body.append(h("table", { class: "proftable" }, h("thead", {}, h("tr", {}, h("th", {}, "Package"), h("th", {}, "Execute"), h("th", {}, "Verify"), h("th", {}, h("span", { class: "visually-hidden" }, "Remove")))), h("tbody", {}, rows)));
    } else {
      body.append(h("p", { class: "muted" }, "Every Package uses the role profiles above."));
    }
    const free = s.packages.filter((name) => !s.overrides.some((o) => o.package === name));
    if (free.length) {
      const select = h("select", { id: "ov-add", "aria-label": "Package" }, free.map((name) => h("option", { value: name }, name)));
      body.append(h("div", { class: "addrow" }, select, h("button", { class: "btn", type: "button", onclick: () => {
        s.overrides.push({ package: select.value, execute: "", verify: "" });
        draw(`#ov-${s.overrides.length - 1}-execute`);
      } }, "Give this Package its own profiles")));
    }
    if (focus) body.querySelector(focus)?.focus();
  };
  draw();
  box.append(body, h("p", { class: "note" }, "An empty field uses the role's profile above. A Package whose fields are both empty keeps no entry."));
  return box;
}

// team chooses between the product's default team and an own team of
// helper roles below every Manager.
function team(s, committed) {
  const box = h("fieldset", { class: "plate" }, h("legend", {}, "Team below each Manager"));
  const body = h("div", {});
  const own = h("input", { type: "radio", name: "team", id: "team-own", checked: Boolean(s.team) });
  const product = h("input", { type: "radio", name: "team", id: "team-default", checked: !s.team });
  const draw = () => {
    clear(body);
    if (!s.team) {
      body.append(h("p", { class: "note" }, "Every Manager works with the product's default team. Choosing an own team replaces it."));
      return;
    }
    s.team.helpers = s.team.helpers || [];
    body.append(h("div", { class: "grid4" }, field("Active helpers at most", s.team, "maxHelpers", { type: "number", hint: "at least 1 when there are helper roles" })));
    const list = h("ol", { class: "helpers" });
    s.team.helpers.forEach((helper, i) => list.append(helperRow(s.team, helper, i, draw)));
    body.append(list, h("button", { class: "btn", type: "button", onclick: () => {
      s.team.helpers.push({ role: "", profile: "", writes: false, instructions: [] });
      draw();
      body.querySelector(".helpers li:last-child input")?.focus();
    } }, "Add a helper role"));
    body.append(h("p", { class: "note" }, "A role name uses lowercase letters, digits and hyphens; manager and reviewer are reserved. A helper without a profile uses the Execute profile. Instructions are repository paths of instruction files, separated by commas."));
  };
  own.addEventListener("change", () => {
    s.team = committed.team ? structuredClone(committed.team) : { maxHelpers: 1, helpers: [{ role: "worker", profile: "", writes: true, instructions: [] }] };
    draw();
  });
  product.addEventListener("change", () => {
    s.team = null;
    draw();
  });
  box.append(h("div", { class: "choice", role: "radiogroup", "aria-label": "Team" },
    h("label", { for: "team-default" }, product, "Product default team"),
    h("label", { for: "team-own" }, own, "Own team")), body);
  draw();
  return box;
}

function helperRow(t, helper, i, redraw) {
  const id = `helper-${i}`;
  helper.instructions = helper.instructions || [];
  const writes = h("input", { type: "checkbox", id: `${id}-writes`, checked: helper.writes });
  writes.addEventListener("change", () => { helper.writes = writes.checked; });
  const instr = h("input", { id: `${id}-instr`, value: helper.instructions.join(", "), placeholder: "none, or docs/role.md, …" });
  instr.addEventListener("input", () => { helper.instructions = instr.value.split(",").map((x) => x.trim()).filter(Boolean); });
  return h("li", { class: "helper" },
    field("Role", helper, "role", { id: `${id}-role` }),
    field("Profile", helper, "profile", { id: `${id}-profile`, placeholder: "Execute profile" }),
    h("div", { class: "field" }, h("label", { for: `${id}-instr` }, "Instructions"), instr),
    h("div", { class: "field check" }, writes, h("label", { for: `${id}-writes` }, "Writes files")),
    h("button", { class: "btn", type: "button", "aria-label": `Remove helper role ${helper.role || i + 1}`, onclick: () => {
      t.helpers.splice(i, 1);
      redraw();
    } }, "Remove"));
}
