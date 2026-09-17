import { useEffect, useRef, useState } from "react";
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
import { TurnstileWidget, type TurnstileHandle } from "@features/auth/TurnstileWidget";

const schema = z
  .object({
    // Optional: an account can now be created without an invitation. A token
    // that IS supplied still has to be valid — the server refuses a bad one
    // rather than quietly handing out a lesser account.
    inviteToken: z.string().optional(),
    username: z
      .string()
      .regex(/^[A-Za-z0-9_]{3,32}$/, "3–32 символа: буквы, цифры, подчёркивание"),
    // What people see and search for — unlike the login, letters only, and
    // unique ignoring case.
    displayName: displayNameSchema,
    password: z
      .string()
      .min(12, "Минимум 12 символов")
      .max(128, "Не более 128 символов")
      .regex(/[A-Za-z]/, "Нужна хотя бы одна буква")
      .regex(/[0-9]/, "Нужна хотя бы одна цифра"),
    confirm: z.string(),
  })
  .refine((d) => d.password === d.confirm, { path: ["confirm"], message: "Пароли не совпадают" });

type Form = z.infer<typeof schema>;

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
      toast.error("На этом сервере регистрация только по приглашению");
      return;
    }
    if (siteKey && !captchaToken) {
      toast.error(
        captchaBroken
          ? "Проверка не загрузилась. Обновите страницу или проверьте соединение"
          : "Секунду — идёт проверка, что вы не робот",
      );
      return;
    }
    try {
      await registerUser(token, data.username, normalizeDisplayName(data.displayName), data.password, captchaToken ?? "");
      toast.success("Аккаунт создан");
      navigate("/", { replace: true });
    } catch (e) {
      // The server has seen this token: whatever went wrong, it will not
      // accept it twice.
      captcha.current?.reset();
      if (e instanceof ApiError && (e.code === "CAPTCHA_FAILED" || e.code === "CAPTCHA_UNAVAILABLE")) {
        toast.error(
          e.code === "CAPTCHA_FAILED"
            ? "Не удалось подтвердить, что вы не робот. Попробуйте ещё раз"
            : "Проверка временно недоступна, попробуйте через минуту",
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
          ? "Код приглашения недействителен или истёк"
          : e instanceof ApiError && e.status === 409
            ? "Имя пользователя уже занято"
            : "Не удалось зарегистрироваться";
      toast.error(msg);
    }
  };

  return (
    <AuthLayout subtitle={openRegistration === false ? "Регистрация по приглашению" : "Создание аккаунта"}>
      <form className="auth-form" onSubmit={handleSubmit(onSubmit)}>
        <Input
          label="Код приглашения"
          // Says outright that the field can be left alone: an empty box under
          // a label reads as something you are missing.
          placeholder={openRegistration === false ? "Обязательно на этом сервере" : "Не обязательно"}
          error={errors.inviteToken?.message}
          {...register("inviteToken")}
        />
        <Input
          label="Имя пользователя"
          placeholder="username"
          autoComplete="username"
          error={errors.username?.message}
          {...register("username")}
        />
        <Input
          label="Имя"
          placeholder="Анна Смирнова"
          autoComplete="name"
          error={errors.displayName?.message}
          {...register("displayName")}
        />
        <Input
          label="Пароль"
          type="password"
          autoComplete="new-password"
          error={errors.password?.message}
          {...register("password")}
        />
        <Input
          label="Повторите пароль"
          type="password"
          autoComplete="new-password"
          error={errors.confirm?.message}
          {...register("confirm")}
        />
        {siteKey && (
          <TurnstileWidget
            ref={captcha}
            siteKey={siteKey}
            onToken={(t) => {
              setCaptchaToken(t);
              if (t) setCaptchaBroken(false);
            }}
            onError={() => setCaptchaBroken(true)}
          />
        )}
        <Button type="submit" block loading={isSubmitting}>
          Создать аккаунт
        </Button>
      </form>
      <p className="auth-footer">
        Уже есть аккаунт?{" "}
        <Link to="/login" className="auth-link">
          Войти
        </Link>
      </p>
    </AuthLayout>
  );
}
