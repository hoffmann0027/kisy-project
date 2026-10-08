import { useMemo } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useNavigate, Link } from "react-router-dom";
import { AuthLayout } from "./AuthLayout";
import { Button, Input, toast } from "@shared/ui";
import { useAuthStore } from "@shared/store/auth";
import { ApiError } from "@shared/api/envelope";
import { t } from "@shared/i18n";

// Built at render, not at import: the messages are in the language on screen.
const makeSchema = () =>
  z.object({
    username: z.string().min(1, t("account.login.usernameRequired")),
    password: z.string().min(1, t("account.login.passwordRequired")),
  });

type Form = z.infer<ReturnType<typeof makeSchema>>;

export function LoginPage() {
  const login = useAuthStore((s) => s.login);
  const navigate = useNavigate();
  const schema = useMemo(makeSchema, []);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<Form>({ resolver: zodResolver(schema) });

  const onSubmit = async (data: Form) => {
    try {
      await login(data.username, data.password);
      navigate("/", { replace: true });
    } catch (e) {
      const msg =
        e instanceof ApiError && e.code === "AUTH_INVALID_CREDENTIALS"
          ? t("account.login.invalidCredentials")
          : e instanceof ApiError && e.status === 429
            ? t("account.login.tooManyAttempts")
            : t("account.login.failed");
      toast.error(msg);
    }
  };

  return (
    <AuthLayout subtitle={t("account.login.subtitle")}>
      <form className="auth-form" onSubmit={handleSubmit(onSubmit)}>
        <Input
          label={t("account.fields.username")}
          placeholder="username"
          autoFocus
          autoComplete="username"
          error={errors.username?.message}
          {...register("username")}
        />
        <Input
          label={t("account.fields.password")}
          type="password"
          placeholder="••••••••••••"
          autoComplete="current-password"
          error={errors.password?.message}
          {...register("password")}
        />
        <Button type="submit" block loading={isSubmitting}>
          {t("account.login.submit")}
        </Button>
      </form>
      <p className="auth-footer">
        {t("account.login.haveInvite")}{" "}
        <Link to="/register" className="auth-link">
          {t("account.login.toRegister")}
        </Link>
      </p>
    </AuthLayout>
  );
}
