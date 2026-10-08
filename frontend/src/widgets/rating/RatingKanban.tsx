import { useMemo, useState, type ReactNode } from "react";
import { Avatar, Button, toast } from "@shared/ui";
import { formatKopecks, parseRublesToKopecks } from "@shared/lib/money";
import { t } from "@shared/i18n";
import type { RatingBoard, RatingProject, RatingTask } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { useRatingMutations } from "@entities/rating/queries";

type Mutations = ReturnType<typeof useRatingMutations>;

function LevelBadge({ level }: { level: number }) {
  return (
    <span className="rating-diff rating-diff--level" title={t("work.rating.levelBadgeTitle", { level })}>
      {t("work.rating.levelShort", { level })}
    </span>
  );
}

// LevelControl shows the project's access level. For the CEO it is an inline
// dropdown that changes the level at any stage (any column); everyone else
// sees a read-only badge.
function LevelControl({ project, m, isCEO }: { project: RatingProject; m: Mutations; isCEO: boolean }) {
  if (!isCEO) return <LevelBadge level={project.minLevel} />;
  return (
    <select
      className="rating-level-select"
      title={t("work.rating.accessLevelHint")}
      value={project.minLevel}
      onClick={(e) => e.stopPropagation()}
      onChange={(e) => {
        const minLevel = Number(e.target.value);
        if (minLevel === project.minLevel) return;
        m.setProjectLevel.mutate(
          { id: project.id, minLevel },
          { onError: () => toast.error(t("work.rating.levelChangeFailed")) },
        );
      }}
    >
      {Array.from({ length: 10 }, (_, i) => i + 1).map((lvl) => (
        <option key={lvl} value={lvl}>
          {t("work.rating.levelShort", { level: lvl })}
        </option>
      ))}
    </select>
  );
}

interface Props {
  board: RatingBoard;
  m: Mutations;
}

// RatingKanban renders three columns:
//  · «Проекты»    — active projects with their backlog tasks;
//  · «В работе»   — assigned tasks of active projects (in progress / done);
//  · «Завершено»  — projects whose tasks are all done, sorted by profit.
// When the last task of a project is completed, the backend deletes the tasks
// and moves the project card here.
export function RatingKanban({ board, m }: Props) {
  const me = useAuthStore((s) => s.user!);
  const isCEO = me.roleLevel === 1;

  const { active, inProgress, done } = useMemo(() => {
    const active = board.projects.filter((p) => p.status === "active");
    const doneProjects = board.projects
      .filter((p) => p.status === "done")
      .sort((a, b) => b.totalProfitKopecks - a.totalProfitKopecks);
    const inProgress = active.flatMap((p) => p.tasks.filter((task) => task.status !== "backlog"));
    return { active, inProgress, done: doneProjects };
  }, [board]);

  return (
    <div className="rating-board">
      <Column title={t("work.rating.columnProjects")} count={active.length}>
        {isCEO && <CreateProjectForm m={m} />}
        {active.length === 0 && <Empty text={t("work.rating.emptyProjects")} />}
        {active.map((p) => (
          <ProjectCard key={p.id} project={p} m={m} isCEO={isCEO} />
        ))}
      </Column>

      <Column title={t("work.rating.columnInProgress")} count={inProgress.length}>
        {inProgress.length === 0 && <Empty text={t("work.rating.emptyInProgress")} />}
        {inProgress.map((task) => (
          <InProgressCard key={task.id} task={task} m={m} mine={task.assignee?.id === me.id} isCEO={isCEO} />
        ))}
      </Column>

      <Column title={t("work.rating.columnDone")} count={done.length}>
        {done.length === 0 && <Empty text={t("work.rating.emptyDone")} />}
        {done.map((p) => (
          <DoneProjectCard key={p.id} project={p} m={m} isCEO={isCEO} />
        ))}
      </Column>
    </div>
  );
}

function Column({ title, count, children }: { title: string; count: number; children: ReactNode }) {
  return (
    <section className="rating-col">
      <header className="rating-col__head">
        <span>{title}</span>
        <span className="rating-col__count">{count}</span>
      </header>
      <div className="rating-col__body">{children}</div>
    </section>
  );
}

function Empty({ text }: { text: string }) {
  return <div className="rating-empty">{text}</div>;
}

function ProjectCard({ project, m, isCEO }: { project: RatingProject; m: Mutations; isCEO: boolean }) {
  const [expanded, setExpanded] = useState(false);
  const [taskTitle, setTaskTitle] = useState("");
  const [addingTask, setAddingTask] = useState(false);
  const backlog = project.tasks.filter((task) => task.status === "backlog");
  const activeTasks = project.tasks.length;

  const addTask = () => {
    const title = taskTitle.trim();
    if (!title) return;
    m.createTask.mutate(
      { projectId: project.id, title },
      {
        onSuccess: () => {
          setTaskTitle("");
          setAddingTask(false);
        },
        onError: () => toast.error(t("work.rating.addTaskFailed")),
      },
    );
  };

  const remove = () => {
    if (!window.confirm(t("work.rating.confirmDeleteProject", { title: project.title }))) return;
    m.deleteProject.mutate(project.id, { onError: () => toast.error(t("work.rating.deleteProjectFailed")) });
  };

  const take = (taskId: string) =>
    m.assign.mutate(taskId, { onError: () => toast.error(t("work.rating.taskTaken")) });

  return (
    <div className="rating-card">
      <div className="rating-card__header-row">
        <button className="rating-card__header" onClick={() => setExpanded((v) => !v)}>
          <span className={"rating-chevron" + (expanded ? " rating-chevron--open" : "")}>▸</span>
          <span className="rating-card__title">{project.title}</span>
          <span className="rating-card__count-chip" title={t("work.rating.taskCount")}>
            {activeTasks}
          </span>
        </button>
        <LevelControl project={project} m={m} isCEO={isCEO} />
      </div>

      {expanded && (
        <div className="rating-card__expand">
          {project.description && <div className="rating-card__desc">{project.description}</div>}

          <div className="rating-tasks">
            {backlog.map((task) => (
              <div key={task.id} className="rating-task">
                <span className="rating-task__title">{task.title}</span>
                <div className="rating-inline">
                  <Button variant="secondary" onClick={() => take(task.id)} loading={m.assign.isPending}>
                    {t("work.rating.take")}
                  </Button>
                  {isCEO && (
                    <button
                      className="rating-x"
                      title={t("work.rating.deleteTask")}
                      onClick={() => m.deleteTask.mutate(task.id)}
                    >
                      ✕
                    </button>
                  )}
                </div>
              </div>
            ))}
            {backlog.length === 0 && <div className="rating-tasks__none">{t("work.rating.allTasksTaken")}</div>}
          </div>

          {isCEO && (
            <div className="rating-card__actions">
              {addingTask ? (
                <div className="rating-inline">
                  <input
                    className="ui-input"
                    placeholder={t("work.rating.taskTitlePlaceholder")}
                    autoFocus
                    value={taskTitle}
                    onChange={(e) => setTaskTitle(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && addTask()}
                  />
                  <Button variant="primary" onClick={addTask} loading={m.createTask.isPending}>
                    +
                  </Button>
                </div>
              ) : (
                <button className="rating-link" onClick={() => setAddingTask(true)}>
                  {t("work.rating.addTask")}
                </button>
              )}
              <button className="rating-link rating-link--danger" onClick={remove}>
                {t("work.rating.deleteProject")}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function InProgressCard({
  task,
  m,
  mine,
  isCEO,
}: {
  task: RatingTask;
  m: Mutations;
  mine: boolean;
  isCEO: boolean;
}) {
  const step = (delta: number) => {
    const next = Math.max(0, Math.min(100, task.progress + delta));
    if (next === task.progress) return;
    m.setProgress.mutate(
      { taskId: task.id, progress: next },
      { onError: () => toast.error(t("work.rating.progressOnlyAssignee")) },
    );
  };

  return (
    <div className="rating-card">
      <div className="rating-card__project">{task.projectTitle}</div>
      <div className="rating-card__title">{task.title}</div>
      {task.assignee && (
        <div className="rating-assignee">
          <Avatar name={task.assignee.displayName} url={task.assignee.avatarUrl} size={24} />
          <span>{task.assignee.displayName}</span>
        </div>
      )}
      <div className="rating-progress">
        <div className="rating-progress__bar">
          <div className="rating-progress__fill" style={{ width: `${task.progress}%` }} />
        </div>
        <span className="rating-progress__pct">{task.progress}%</span>
      </div>
      <div className="rating-progress__ctl">
        {mine && (
          <>
            <Button variant="ghost" onClick={() => step(-10)} disabled={task.progress === 0 || m.setProgress.isPending}>
              −10%
            </Button>
            <Button variant="secondary" onClick={() => step(10)} loading={m.setProgress.isPending}>
              +10%
            </Button>
          </>
        )}
        {(mine || isCEO) && (
          <button className="rating-link" onClick={() => m.returnTask.mutate(task.id)}>
            {t("work.rating.returnTask")}
          </button>
        )}
        {isCEO && (
          <button className="rating-link rating-link--danger" onClick={() => m.deleteTask.mutate(task.id)}>
            {t("work.rating.delete")}
          </button>
        )}
      </div>
    </div>
  );
}

function DoneProjectCard({ project, m, isCEO }: { project: RatingProject; m: Mutations; isCEO: boolean }) {
  const [open, setOpen] = useState(false);
  const [income, setIncome] = useState("");
  const [expense, setExpense] = useState("");
  const [note, setNote] = useState("");

  const submit = () => {
    const inc = parseRublesToKopecks(income);
    const exp = parseRublesToKopecks(expense);
    if (inc === null || exp === null) {
      toast.error(t("work.rating.invalidAmounts"));
      return;
    }
    if (inc === 0 && exp === 0) {
      toast.error(t("work.rating.noAmounts"));
      return;
    }
    m.addFinance.mutate(
      { projectId: project.id, incomeKopecks: inc, expenseKopecks: exp, note: note.trim() || undefined },
      {
        onSuccess: () => {
          setIncome("");
          setExpense("");
          setNote("");
          setOpen(false);
        },
        onError: () => toast.error(t("work.rating.addFinanceFailed")),
      },
    );
  };

  const remove = () => {
    if (!window.confirm(t("work.rating.confirmDeleteDoneProject", { title: project.title }))) return;
    m.deleteProject.mutate(project.id, { onError: () => toast.error(t("work.rating.deleteProjectFailed")) });
  };

  return (
    <div className="rating-card">
      <div className="rating-card__top">
        <div className="rating-card__title">{project.title}</div>
        <LevelControl project={project} m={m} isCEO={isCEO} />
      </div>
      <div className="rating-profit">
        <span>{t("work.rating.profitAllTime")}</span>
        <strong className={project.totalProfitKopecks < 0 ? "rating-profit--neg" : "rating-profit--pos"}>
          {formatKopecks(project.totalProfitKopecks)}
        </strong>
      </div>
      {isCEO &&
        (open ? (
          <div className="rating-finance">
            <input
              className="ui-input"
              placeholder={t("work.rating.incomePlaceholder")}
              inputMode="decimal"
              value={income}
              onChange={(e) => setIncome(e.target.value)}
            />
            <input
              className="ui-input"
              placeholder={t("work.rating.expensePlaceholder")}
              inputMode="decimal"
              value={expense}
              onChange={(e) => setExpense(e.target.value)}
            />
            <input
              className="ui-input"
              placeholder={t("work.rating.notePlaceholder")}
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
            <div className="rating-inline">
              <Button variant="ghost" onClick={() => setOpen(false)}>
                {t("work.rating.cancel")}
              </Button>
              <Button variant="primary" onClick={submit} loading={m.addFinance.isPending}>
                {t("work.rating.submitFinance")}
              </Button>
            </div>
          </div>
        ) : (
          <div className="rating-card__actions">
            <button className="rating-link" onClick={() => setOpen(true)}>
              {t("work.rating.addFinance")}
            </button>
            <button className="rating-link rating-link--danger" onClick={remove}>
              {t("work.rating.delete")}
            </button>
          </div>
        ))}
    </div>
  );
}

function CreateProjectForm({ m }: { m: Mutations }) {
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [minLevel, setMinLevel] = useState(10);
  const [description, setDescription] = useState("");

  const submit = () => {
    const name = title.trim();
    if (!name) return;
    m.createProject.mutate(
      { title: name, minLevel, description: description.trim() || undefined },
      {
        onSuccess: () => {
          setTitle("");
          setDescription("");
          setMinLevel(10);
          setOpen(false);
        },
        onError: () => toast.error(t("work.rating.createProjectFailed")),
      },
    );
  };

  if (!open) {
    return (
      <button className="rating-add" onClick={() => setOpen(true)}>
        {t("work.rating.newProject")}
      </button>
    );
  }

  return (
    <div className="rating-card rating-create">
      <input
        className="ui-input"
        placeholder={t("work.rating.projectTitlePlaceholder")}
        autoFocus
        value={title}
        onChange={(e) => setTitle(e.target.value)}
      />
      <textarea
        className="ui-input"
        placeholder={t("work.rating.descriptionPlaceholder")}
        rows={2}
        value={description}
        onChange={(e) => setDescription(e.target.value)}
      />
      <label className="rating-level-label">
        {t("work.rating.accessLevelHint")}
        <select className="ui-input" value={minLevel} onChange={(e) => setMinLevel(Number(e.target.value))}>
          {Array.from({ length: 10 }, (_, i) => i + 1).map((lvl) => (
            <option key={lvl} value={lvl}>
              {t("work.rating.levelOption", { level: lvl })}
            </option>
          ))}
        </select>
      </label>
      <div className="rating-inline">
        <Button variant="ghost" onClick={() => setOpen(false)}>
          {t("work.rating.cancel")}
        </Button>
        <Button variant="primary" onClick={submit} loading={m.createProject.isPending}>
          {t("work.rating.create")}
        </Button>
      </div>
    </div>
  );
}
