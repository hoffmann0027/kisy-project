import { forwardRef, type InputHTMLAttributes } from "react";
import { cn } from "@shared/lib/cn";

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

export const Input = forwardRef<HTMLInputElement, Props>(function Input(
  { label, error, className, id, ...rest },
  ref,
) {
  return (
    <div className="ui-field">
      {label && (
        <label className="ui-field__label" htmlFor={id}>
          {label}
        </label>
      )}
      <input ref={ref} id={id} className={cn("ui-input", error && "ui-input--error", className)} {...rest} />
      {error && <span className="ui-field__error">{error}</span>}
    </div>
  );
});
