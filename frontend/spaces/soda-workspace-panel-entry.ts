// The workspace panel is the retained half of the one shared Workspace
// owner (NFX-D02/H-OWNER). It shares the spaces page entry and mount
// implementation; the host selects the single panel instance and drives
// it through the returned MountHandle. This module re-exports that mount
// for existing panel fixture imports.
export {mount} from './soda-spaces-entry.js';
