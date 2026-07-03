import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useNavigate, Link } from "react-router-dom";
import { AuthLayout } from "./AuthLayout";
import { Button, Input, toast } from "@shared/ui";
import { useAuthStore } from "@shared/store/auth";
import { ApiError } from "@shared/api/envelope";

const schema = z.object({
  username: z.string().min(1, "Введите имя пользователя"),
  password: z.string().min(1, "Введите пароль"),
});

type Form = z.infer<typeof schema>;

export function LoginPage() {
  const login = useAuthStore((s) => s.login);
  const navigate = useNavigate();
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
          ? "Неверное имя пользователя или пароль"
          : e instanceof ApiError && e.status === 429
            ? "Слишком много попыток. Попробуйте позже"
            : "Не удалось войти";
      toast.error(msg);
    }
  };

  return (
    <AuthLayout subtitle="Корпоративный мессенджер">
      <form className="auth-form" onSubmit={handleSubmit(onSubmit)}>
        <Input
          label="Имя пользователя"
          placeholder="username"
          autoFocus
          autoComplete="username"
          error={errors.username?.message}
          {...register("username")}
        />
        <Input
          label="Пароль"
          type="password"
          placeholder="••••••••••••"
          autoComplete="current-password"
          error={errors.password?.message}
          {...register("password")}
        />
        <Button type="submit" block loading={isSubmitting}>
          Войти
        </Button>
      </form>
      <p className="auth-footer">
        Есть код приглашения?{" "}
        <Link to="/register" className="auth-link">
          Зарегистрироваться
        </Link>
      </p>
    </AuthLayout>
  );
}
