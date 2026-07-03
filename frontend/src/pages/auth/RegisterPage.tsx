import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useNavigate, useSearchParams, Link } from "react-router-dom";
import { AuthLayout } from "./AuthLayout";
import { Button, Input, toast } from "@shared/ui";
import { useAuthStore } from "@shared/store/auth";
import { ApiError } from "@shared/api/envelope";

const schema = z
  .object({
    inviteToken: z.string().min(1, "Введите код приглашения"),
    username: z
      .string()
      .regex(/^[A-Za-z0-9_]{3,32}$/, "3–32 символа: буквы, цифры, подчёркивание"),
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
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: { inviteToken: params.get("token") ?? "" },
  });

  const onSubmit = async (data: Form) => {
    try {
      await registerUser(data.inviteToken, data.username, data.password);
      toast.success("Аккаунт создан");
      navigate("/", { replace: true });
    } catch (e) {
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
    <AuthLayout subtitle="Регистрация по приглашению">
      <form className="auth-form" onSubmit={handleSubmit(onSubmit)}>
        <Input
          label="Код приглашения"
          placeholder="Токен от администратора"
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
