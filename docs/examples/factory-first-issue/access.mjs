// Starting behavior: only active HR users can view a task.
export function canViewTask(user, _task) {
  return user.active === true && user.role === "hr";
}
