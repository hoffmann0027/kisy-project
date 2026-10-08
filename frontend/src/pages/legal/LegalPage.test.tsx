import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { applyDictionary, loadDictionary } from "@shared/i18n";
import { ru } from "@shared/i18n/locales/ru";
import { PrivacyPage } from "./LegalPage";

// A translated legal document says it is a translation, and the binding
// Russian text is one tap away; in Russian there is nothing to say.

afterEach(() => applyDictionary("ru", ru));

const renderPage = () =>
  render(
    <MemoryRouter>
      <PrivacyPage />
    </MemoryRouter>,
  );

describe("the privacy policy", () => {
  it("in English: marked as a translation, with the original a tap away", async () => {
    applyDictionary("en", await loadDictionary("en"));
    renderPage();
    expect(await screen.findByText(/The Russian version is the legally binding one/)).toBeTruthy();
    fireEvent.click(screen.getByText("Show the original in Russian"));
    expect(await screen.findByText("Коротко")).toBeTruthy();
    expect(screen.getByText("Show the translation")).toBeTruthy();
  });

  it("in Russian: the original, with no note", () => {
    renderPage();
    expect(screen.getByText("Коротко")).toBeTruthy();
    expect(screen.queryByText(/Юридическую силу имеет/)).toBeNull();
  });
});
