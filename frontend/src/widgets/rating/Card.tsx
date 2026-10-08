import type { ReactNode } from "react";
import { cn } from "@shared/lib/cn";

interface Props {
  title: string;
  /** Controls on the right of the title: tabs, a range, a link. */
  aside?: ReactNode;
  className?: string;
  children: ReactNode;
}

/** One panel of the dashboard: a title row and whatever it shows. */
export function Card({ title, aside, className, children }: Props) {
  return (
    <section className={cn("rc", className)} aria-label={title}>
      <header className="rc__head">
        <h2 className="rc__title">{title}</h2>
        {aside}
      </header>
      {children}
    </section>
  );
}

interface TabsProps<T extends string> {
  value: T;
  options: { value: T; label: string }[];
  onChange: (value: T) => void;
  label: string;
}

/** A row of small toggles inside a card's title row. */
export function CardTabs<T extends string>({ value, options, onChange, label }: TabsProps<T>) {
  return (
    <div className="rc__tabs" role="group" aria-label={label}>
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          className={cn("rc__tab", o.value === value && "is-active")}
          aria-pressed={o.value === value}
          onClick={() => onChange(o.value)}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

/** What a panel says when it has nothing to show yet. */
export function CardEmpty({ children }: { children: ReactNode }) {
  return <div className="rc__empty">{children}</div>;
}
