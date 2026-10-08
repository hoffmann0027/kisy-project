import { useEffect, useMemo, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useNavigate, useSearchParams, Link } from "react-router-dom";
import { AuthLayout } from "./AuthLayout";
import { Button, Input, toast } from "@shared/ui";
import { useAuthStore } from "@shared/store/auth";
import { authApi } from "@shared/api/endpoints";
import { ApiError } from "@shared/api/envelope";
import { displayNameErrorMessage, displayNameSchema, normalizeDisplayName } from "@shared/lib/displayName";
import { passwordProblem, passwordRuleText } from "@shared/lib/password";
import { TurnstileWidget, type TurnstileHandle } from "@features/auth/TurnstileWidget";
import { t } from "@shared/i18n";

// Built at render, not at import: the messages are in the language on screen.
const makeSchema = () =>
  z
    .object({
      // Optional: an account can now be created without an invitation. A token
      // that IS supplied still has to be valid — the server refuses a bad one
      // rather than quietly handing out a lesser account.
      inviteToken: z.string().optional(),
      username: z
        .string()
        .regex(/^[A-Za-z0-9_]{3,32}$/, t("account.register.usernameRule")),
      // What people see and search for — unlike the login, letters only, and
      // unique ignoring case.
      displayName: displayNameSchema,
      // The server's rule, not a stricter one: /[A-Za-z]/ here refused a
      // Cyrillic password the API accepts (audit D-14).
      password: z.string().superRefine((value, ctx) => {
        const problem = passwordProblem(value);
        if (problem) ctx.addIssue({ code: z.ZodIssueCode.custom, message: problem });
      }),
      confirm: z.string(),
    })
    .refine((d) => d.password === d.confirm, { path: ["confirm"], message: t("account.password.mismatch") });

type Form = z.infer<ReturnType<typeof makeSchema>>;

export function RegisterPage() {
  const registerUser = useAuthStore((s) => s.register);
  const navigate = useNavigate();
  const [params] = useSearchParams();
  // Whether this deployment accepts accounts without an invitation.
  //
  // Not "hide the page when closed": an invited person still needs this form.
  // What closing changes is whether the code is optional — and saying so up
  // front beats letting someone fill in four fields for a refusal.
  const [openRegistration, setOpenRegistration] = useState<boolean | null>(null);
  // Turnstile: the site key comes with the policy (empty — no check on this
  // deployment). The token is single use, so every refused attempt resets it.
  const [siteKey, setSiteKey] = useState("");
  const [captchaToken, setCaptchaToken] = useState<string | null>(null);
  const [captchaBroken, setCaptchaBroken] = useState(false);
  const captcha = useRef<TurnstileHandle>(null);
  const schema = useMemo(makeSchema, []);

  useEffect(() => {
    let dropped = false;
    void authApi
      .registrationPolicy()
      .then((p) => {
        if (dropped) return;
        setOpenRegistration(p.open);
        setSiteKey(p.turnstileSiteKey ?? "");
      })
      // Unreachable server: assume the stricter of the two, so the form never
      // promises something the deployment does not allow.
      .catch(() => !dropped && setOpenRegistration(false));
    return () => {
      dropped = true;
    };
  }, []);
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: { inviteToken: params.get("token") ?? "" },
  });

  const onSubmit = async (data: Form) => {
    const token = (data.inviteToken ?? "").trim();
    if (!token && openRegistration === false) {
      toast.error(t("account.register.inviteOnly"));
      return;
    }
    if (siteKey && !captchaToken) {
      toast.error(
        captchaBroken
          ? t("account.register.captchaBroken")
          : t("account.register.captchaPending"),
      );
      return;
    }
    try {
      await registerUser(token, data.username, normalizeDisplayName(data.displayName), data.password, captchaToken ?? "");
      toast.success(t("account.register.created"));
      navigate("/", { replace: true });
    } catch (e) {
      // The server has seen this token: whatever went wrong, it will not
      // accept it twice.
      captcha.current?.reset();
      if (e instanceof ApiError && (e.code === "CAPTCHA_FAILED" || e.code === "CAPTCHA_UNAVAILABLE")) {
        toast.error(
          e.code === "CAPTCHA_FAILED"
            ? t("account.register.captchaFailed")
            : t("account.register.captchaUnavailable"),
        );
        return;
      }
      // A name problem belongs under the name field, not in a toast.
      const nameProblem = displayNameErrorMessage(e);
      if (nameProblem) {
        setError("displayName", { message: nameProblem });
        return;
      }
      const msg =
        e instanceof ApiError && e.code === "AUTH_INVALID_TOKEN"
          ? t("account.register.invalidInvite")
          : e instanceof ApiError && e.status === 409
            ? t("account.register.usernameTaken")
            : t("account.register.failed");
      toast.error(msg);
    }
  };

  return (
    <AuthLayout
      subtitle={openRegistration === false ? t("account.register.subtitleInviteOnly") : t("account.register.subtitle")}
    >
      <form className="auth-form" onSubmit={handleSubmit(onSubmit)}>
        <Input
          label={t("account.register.inviteCode")}
          // Says outright that the field can be left alone: an empty box under
          // a label reads as something you are missing.
          placeholder={
            openRegistration === false ? t("account.register.inviteRequired") : t("account.register.inviteOptional")
          }
          error={errors.inviteToken?.message}
          {...register("inviteToken")}
        />
        <Input
          label={t("account.fields.username")}
          placeholder="username"
          autoComplete="username"
          error={errors.username?.message}
          {...register("username")}
        />
        <Input
          label={t("account.register.name")}
          placeholder={t("account.fields.namePlaceholder")}
          autoComplete="name"
          error={errors.displayName?.message}
          {...register("displayName")}
        />
        <Input
          label={t("account.fields.password")}
          hint={passwordRuleText()}
          type="password"
          autoComplete="new-password"
          error={errors.password?.message}
          {...register("password")}
        />
        <Input
          label={t("account.register.confirmPassword")}
          type="password"
          autoComplete="new-password"
          error={errors.confirm?.message}
          {...register("confirm")}
        />
        {siteKey && (
          <TurnstileWidget
            ref={captcha}
            siteKey={siteKey}
            onToken={(token) => {
              setCaptchaToken(token);
              if (token) setCaptchaBroken(false);
            }}
            onError={() => setCaptchaBroken(true)}
          />
        )}
        <Button type="submit" block loading={isSubmitting}>
          {t("account.register.submit")}
        </Button>
      </form>
      <p className="auth-footer">
        {t("account.register.haveAccount")}{" "}
        <Link to="/login" className="auth-link">
          {t("account.register.toLogin")}
        </Link>
      </p>
    </AuthLayout>
  );
}
