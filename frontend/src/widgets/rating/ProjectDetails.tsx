import { useState } from "react";
import { Avatar, Button, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { t } from "@shared/i18n";
import { formatKopecks } from "@shared/lib/money";
import { formatDay } from "@shared/lib/format";
import type { RatingProject, RatingTask } from "@shared/api/types";
import type { useRatingMutations } from "@entities/rating/queries";
import { FinanceDialog } from "./FinanceDialog";

type Mutations = ReturnType<typeof useRatingMutations>;

interface Props {
  project: RatingProject;
  m: Mutations;
  isCEO: boolean;
  meId: string;
}

/**
 * What opens under a project's row: its description and tasks, and the
 * controls each person has — take a task, move your own along, return it;
 * for the CEO also add tasks, record money, set the level, delete.
 */
export function ProjectDetails({ project, m, isCEO, meId }: Props) {
  const [taskTitle, setTaskTitle] = useState("");
  const [addingTask, setAddingTask] = useState(false);
  const [finance, setFinance] = useState(false);
  const done = project.status === "done";

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
    const question = done ? "work.rating.confirmDeleteDoneProject" : "work.rating.confirmDeleteProject";
    if (!window.confirm(t(question, { title: project.title }))) return;
    m.deleteProject.mutate(project.id, { onError: () => toast.error(t("work.rating.deleteProjectFailed")) });
  };

  const changeLevel = (minLevel: number) => {
    if (minLevel === project.minLevel) return;
    m.setProjectLevel.mutate({ id: project.id, minLevel }, { onError: () => toast.error(t("work.rating.levelChangeFailed")) });
  };

  return (
    <div className="rt__details">
      {project.description && <p className="rt__desc-full">{project.description}</p>}

      <div className="rt__facts">
        <span className="rt__fact">
          <span className="rt__fact-label">{t("work.rating.levelLabel")}</span>
          {isCEO ? (
            <select
              className="rating-level-select"
              title={t("work.rating.accessLevelHint")}
              value={project.minLevel}
              disabled={m.setProjectLevel.isPending}
              onChange={(e) => changeLevel(Number(e.target.value))}
            >
              {Array.from({ length: 10 }, (_, i) => i + 1).map((lvl) => (
                <option key={lvl} value={lvl}>
                  {t("work.rating.levelShort", { level: lvl })}
                </option>
              ))}
            </select>
          ) : (
            <span className="rchip rchip--level" title={t("work.rating.levelBadgeTitle", { level: project.minLevel })}>
              {t("work.rating.levelShort", { level: project.minLevel })}
            </span>
          )}
        </span>
        <span className="rt__fact">
          <span className="rt__fact-label">{t("work.rating.profitAllTime")}</span>
          <b className={project.totalProfitKopecks < 0 ? "rt__money--out" : "rt__money--in"}>{formatKopecks(project.totalProfitKopecks)}</b>
        </span>
        {project.completedAt && (
          <span className="rt__fact">
            <span className="rt__fact-label">{t("work.rating.statusDone")}</span>
            <b>{formatDay(project.completedAt)}</b>
          </span>
        )}
      </div>

      {!done && (
        <div className="rtasks">
          <div className="rtasks__head">{t("work.rating.tasksTitle")}</div>
          {project.tasks.length === 0 && <div className="rtasks__none">{t("work.rating.noTasks")}</div>}
          {project.tasks.map((task) => (
            <TaskRow key={task.id} task={task} m={m} isCEO={isCEO} mine={task.assignee?.id === meId} />
          ))}
          {project.tasks.length > 0 && project.tasks.every((task) => task.status !== "backlog") && (
            <div className="rtasks__none">{t("work.rating.allTasksTaken")}</div>
          )}
        </div>
      )}

      {isCEO && (
        <div className="rt__ceo">
          {!done &&
            (addingTask ? (
              <div className="rt__addtask">
                <input
                  className="ui-input"
                  placeholder={t("work.rating.taskTitlePlaceholder")}
                  autoFocus
                  value={taskTitle}
                  onChange={(e) => setTaskTitle(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && addTask()}
                />
                <Button variant="primary" onClick={addTask} loading={m.createTask.isPending}>
                  {t("work.rating.create")}
                </Button>
                <Button variant="ghost" onClick={() => setAddingTask(false)}>
                  {t("work.rating.cancel")}
                </Button>
              </div>
            ) : (
              <Button variant="secondary" onClick={() => setAddingTask(true)}>
                <Icon.Plus size={16} /> {t("work.rating.addTask")}
              </Button>
            ))}
          <Button variant="secondary" onClick={() => setFinance(true)}>
            {t("work.rating.addFinance")}
          </Button>
          <Button variant="danger" onClick={remove} loading={m.deleteProject.isPending}>
            {t("work.rating.deleteProject")}
          </Button>
          <FinanceDialog project={project} m={m} open={finance} onClose={() => setFinance(false)} />
        </div>
      )}
    </div>
  );
}

function TaskRow({ task, m, isCEO, mine }: { task: RatingTask; m: Mutations; isCEO: boolean; mine: boolean }) {
  const step = (delta: number) => {
    const next = Math.max(0, Math.min(100, task.progress + delta));
    if (next === task.progress) return;
    m.setProgress.mutate({ taskId: task.id, progress: next }, { onError: () => toast.error(t("work.rating.progressOnlyAssignee")) });
  };
  const take = () => m.assign.mutate(task.id, { onError: () => toast.error(t("work.rating.taskTaken")) });
  const backlog = task.status === "backlog";
  const finished = task.status === "done";

  return (
    <div className={"rtask" + (finished ? " rtask--done" : "")}>
      <span className="rtask__title">{task.title}</span>
      {task.assignee ? (
        <span className="rtask__who">
          <Avatar name={task.assignee.displayName} url={task.assignee.avatarUrl} size={22} />
          <span>{task.assignee.displayName}</span>
        </span>
      ) : (
        <span className="rtask__who rtask__who--none">{t("work.rating.statusOpen")}</span>
      )}
      {!backlog && (
        <span className="rtask__progress">
          <span className="rbar" role="progressbar" aria-valuenow={task.progress} aria-valuemin={0} aria-valuemax={100}>
            <span className="rbar__fill" style={{ width: `${task.progress}%` }} />
          </span>
          <span className="rtask__pct">{task.progress}%</span>
        </span>
      )}
      <span className="rtask__actions">
        {backlog && (
          <Button variant="secondary" onClick={take} loading={m.assign.isPending}>
            {t("work.rating.take")}
          </Button>
        )}
        {mine && !backlog && !finished && (
          <>
            <Button variant="ghost" onClick={() => step(-10)} disabled={task.progress === 0 || m.setProgress.isPending}>
              −10%
            </Button>
            <Button variant="secondary" onClick={() => step(10)} loading={m.setProgress.isPending}>
              +10%
            </Button>
          </>
        )}
        {(mine || isCEO) && !backlog && (
          <Button variant="ghost" onClick={() => m.returnTask.mutate(task.id)}>
            {t("work.rating.returnTask")}
          </Button>
        )}
        {isCEO && (
          <button
            type="button"
            className="ui-icon-btn rtask__delete"
            title={t("work.rating.deleteTask")}
            aria-label={`${t("work.rating.deleteTask")}: ${task.title}`}
            onClick={() => m.deleteTask.mutate(task.id)}
          >
            <Icon.Trash size={16} />
          </button>
        )}
      </span>
    </div>
  );
}
