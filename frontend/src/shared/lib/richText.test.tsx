import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { renderRichText, firstUrl } from "./richText";

function html(node: React.ReactNode): string {
  const { container } = render(<div>{node}</div>);
  return container.innerHTML;
}

describe("renderRichText", () => {
  it("renders bold, italic, strike and inline code", () => {
    expect(html(renderRichText("**b**"))).toContain("<strong>b</strong>");
    expect(html(renderRichText("_i_"))).toContain("<em>i</em>");
    expect(html(renderRichText("~~s~~"))).toContain("<del>s</del>");
    expect(html(renderRichText("`c`"))).toContain('class="rt-code"');
  });

  it("nests formatting (bold containing italic)", () => {
    const out = html(renderRichText("**a _b_ c**"));
    expect(out).toContain("<strong>");
    expect(out).toContain("<em>b</em>");
  });

  it("renders a fenced code block with a copy button", () => {
    const out = html(renderRichText("```\nline1\nline2\n```"));
    expect(out).toContain('class="rt-codeblock"');
    expect(out).toContain("line1\nline2");
    expect(out).toContain("Копировать");
  });

  it("autolinks URLs and highlights mentions", () => {
    const out = html(renderRichText("см https://example.com и @user_1"));
    expect(out).toContain('href="https://example.com"');
    expect(out).toContain('rel="noopener noreferrer nofollow"');
    expect(out).toContain('class="mention">@user_1</span>');
  });

  it("never emits raw HTML from the source (no injection)", () => {
    const out = html(renderRichText("<img src=x onerror=alert(1)> **safe**"));
    // The angle brackets are escaped by React as text, not parsed as a tag.
    expect(out).not.toContain("<img");
    expect(out).toContain("&lt;img");
    expect(out).toContain("<strong>safe</strong>");
  });

  it("leaves plain text untouched", () => {
    expect(html(renderRichText("просто текст"))).toContain("просто текст");
  });
});

describe("firstUrl", () => {
  it("finds the first http(s) URL or null", () => {
    expect(firstUrl("нет ссылок")).toBeNull();
    expect(firstUrl("a https://x.io/p?q=1 b http://y.io")).toBe("https://x.io/p?q=1");
  });
});
