import { forwardRef, useId, type InputHTMLAttributes } from "react";
import { cn } from "@shared/lib/cn";

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

export const Input = forwardRef<HTMLInputElement, Props>(function Input(
  { label, error, className, id, ...rest },
  ref,
) {
  // The label was tied to the input only when a caller passed an id, and
  // almost none do — so most labelled fields in the app were labels in
  // appearance only: a screen reader read an unnamed box, and tapping the
  // caption did not focus anything. A generated id costs nothing and makes the
  // association real everywhere.
  const generated = useId();
  const inputId = id ?? generated;
  const errorId = `${inputId}-error`;

  return (
    <div className="ui-field">
      {label && (
        <label className="ui-field__label" htmlFor={inputId}>
          {label}
        </label>
      )}
      <input
        ref={ref}
        id={inputId}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errorId : undefined}
        className={cn("ui-input", error && "ui-input--error", className)}
        {...rest}
      />
      {error && (
        <span className="ui-field__error" id={errorId}>
          {error}
        </span>
      )}
    </div>
  );
});
