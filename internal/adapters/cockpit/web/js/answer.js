// The answer form for an owner question. Writing is a deliberate two-step
// action: the answer is final, it clarifies within the model and cannot change
// it. A written answer file is not yet an answer the run recorded; the run
// records it when it reads the owner channel, and the record then shows it.
// Drafts live outside the page's DOM, so a redraw never loses them.

import { api } from "./api.js";
import { h, clear, clock } from "./dom.js";

const MAX_BYTES = 16 << 10; // the owner channel's limit, in UTF-8 bytes
const drafts = new Map(); // run/question -> { text, reviewing, written }

// answerInProgress says whether a redraw of root would interrupt the owner:
// an answer form with text, under review or in focus.
export function answerInProgress(root) {
  return [...root.querySelectorAll(".answer textarea")].some((t) => t.value.trim() || t.readOnly || t === document.activeElement);
}

export function answerForm(run, question) {
  const key = `${run}/${question}`;
  const draft = drafts.get(key) || { text: "", reviewing: false, written: null };
  drafts.set(key, draft);
  const box = h("div", { class: "answer" });
  const id = `answer-${run}-${question}`;
  const status = h("p", { class: "note", role: "status", tabindex: -1 });
  if (draft.written) {
    box.append(h("p", { class: "written" }, writtenText(question, draft.written)));
    return box;
  }
  const text = h("textarea", { id, rows: 4, "aria-describedby": `${id}-note` });
  text.value = draft.text;
  text.addEventListener("input", () => { draft.text = text.value; });
  const note = h("p", { class: "note", id: `${id}-note` }, "Answers clarify within the model; they cannot change it. An answer is final. A model Root judges erroneous or contradictory still ends the run.");
  const actions = h("div", { class: "actions" });
  const editing = () => {
    draft.reviewing = false;
    text.readOnly = false;
    const review = h("button", { class: "btn", type: "button" }, "Review the answer");
    review.addEventListener("click", () => {
      const bytes = new TextEncoder().encode(text.value).length;
      if (!text.value.trim()) {
        status.textContent = "Write the answer first.";
        text.focus();
      } else if (bytes > MAX_BYTES) {
        status.textContent = `The answer has ${bytes} bytes; the owner channel takes at most ${MAX_BYTES}.`;
        text.focus();
      } else {
        reviewing();
      }
    });
    clear(actions).append(review);
    status.textContent = "";
  };
  const reviewing = () => {
    draft.reviewing = true;
    text.readOnly = true;
    const confirm = h("button", { class: "btn primary-text", type: "button" }, `Write the answer to ${question}`);
    const back = h("button", { class: "btn", type: "button" }, "Edit");
    clear(actions).append(confirm, back);
    status.textContent = `This writes the answer to run ${run}'s owner channel. It is final.`;
    back.addEventListener("click", () => {
      editing();
      text.focus();
    });
    confirm.addEventListener("click", async () => {
      confirm.disabled = back.disabled = true;
      status.textContent = "Writing…";
      try {
        const res = await api.answer(run, question, text.value);
        draft.written = res;
        text.remove();
        note.remove();
        actions.remove();
        status.textContent = writtenText(question, res);
        status.classList.add("written");
        status.focus();
      } catch (err) {
        confirm.disabled = back.disabled = false;
        status.textContent = `Not written: ${err.message}`;
      }
    });
    confirm.focus();
  };
  if (draft.reviewing) reviewing();
  else editing();
  box.append(h("label", { for: id }, `Answer to ${question}`), text, note, actions, status);
  return box;
}

function writtenText(question, res) {
  return `Answer to ${question} written at ${clock(res.written)} to ${res.path}. The run has not recorded it yet; it does when it reads the owner channel, and the question then shows as answered.`;
}
