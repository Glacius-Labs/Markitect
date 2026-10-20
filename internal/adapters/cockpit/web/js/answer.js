// The answer form for an owner question. Writing is a deliberate two-step
// action: the answer is final, it clarifies within the model and cannot change
// it. A written answer file is not yet an answer the run recorded; the run
// records it when it reads the owner channel, and the record then shows it.

import { api } from "./api.js";
import { h, clear, clock } from "./dom.js";

export function answerForm(run, question, onWritten) {
  const box = h("div", { class: "answer" });
  const id = `answer-${run}-${question}`;
  const text = h("textarea", { id, rows: 4, maxlength: 16384, "aria-describedby": `${id}-note` });
  const note = h("p", { class: "note", id: `${id}-note` }, "Answers clarify within the model; they cannot change it. An answer is final. A model Root judges erroneous or contradictory still ends the run.");
  const status = h("p", { class: "note", role: "status" });
  const write = h("button", { class: "btn", type: "button" }, "Review the answer");
  const actions = h("div", { class: "actions" }, write);
  write.addEventListener("click", () => {
    if (!text.value.trim()) {
      status.textContent = "Write the answer first.";
      text.focus();
      return;
    }
    text.readOnly = true;
    const confirm = h("button", { class: "btn primary-text", type: "button" }, `Write the answer to ${question}`);
    const back = h("button", { class: "btn", type: "button" }, "Edit");
    clear(actions).append(confirm, back);
    status.textContent = `This writes the answer to run ${run}'s owner channel. It is final.`;
    confirm.focus();
    back.addEventListener("click", () => {
      text.readOnly = false;
      clear(actions).append(write);
      status.textContent = "";
      text.focus();
    });
    confirm.addEventListener("click", async () => {
      confirm.disabled = back.disabled = true;
      status.textContent = "Writing…";
      try {
        const res = await api.answer(run, question, text.value);
        clear(box).append(h("p", { class: "written", role: "status" },
          `Answer to ${question} written at ${clock(res.written)} to ${res.path}. The run has not recorded it yet; it does when it reads the owner channel, and the question then shows as answered.`));
        onWritten?.(res);
      } catch (err) {
        confirm.disabled = back.disabled = false;
        status.textContent = `Not written: ${err.message}`;
      }
    });
  });
  box.append(h("label", { for: id }, `Answer to ${question}`), text, note, actions, status);
  return box;
}
