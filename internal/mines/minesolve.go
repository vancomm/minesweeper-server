// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package mines

import (
	"math/rand/v2"
	"strconv"
)

/* ----------------------------------------------------------------------
 * Minesweeper solver, used to ensure the generated grids are
 * solvable without having to take risks.
 */

type solveResult int8

const (
	NA solveResult = iota - 2
	Stalled
	Success
	// values >0 mean given number of perturbations was required
)

func (r solveResult) String() string {
	switch r {
	case NA:
		return "NA"
	case Stalled:
		return "stalled"
	case Success:
		return "success"
	default:
		return strconv.Itoa(int(r)) + " perturbs"
	}
}

/*
panics [AssertionError]

Main solver entry point. You give it a grid of existing
knowledge (-1 for a square known to be a mine, 0-8 for empty
squares with a given number of neighbours, -2 for completely
unknown), plus a function which you can call to open new squares
once you're confident of them. It fills in as much more of the
grid as it can.

Return value is:

  - -1 means deduction stalled and nothing could be done
  - 0 means deduction succeeded fully
  - '>0' means deduction succeeded but some number of perturbation
    steps were required; the exact return value is the number of
    perturb calls.
*/
func mineSolve(
	w, h, n int,
	grid Grid,
	ctx *mineCtx,
	r *rand.Rand,
) solveResult {
	ss := newSetStore()
	nperturbs := 0

	/*
	 * Set up a linked list of squares with known contents, so that
	 * we can process them one by one.
	 */
	std := &celltodo{
		next: make([]int, w*h),
		head: -1,
		tail: -1,
	}

	/*
	 * Initialise that list with all known squares in the input
	 * grid.
	 */
	for y := range h {
		for x := range w {
			i := y*w + x
			if grid[i] != Unknown {
				std.add(i)
			}
		}
	}

	/*
	 * Main deductive loop.
	 */
	for {
		doneSomething := false

		/*
		 * If there are any known squares on the todo list, process
		 * them and construct a set for each.
		 */
		for std.head != -1 {
			i := std.head
			std.head = std.next[i]
			if std.head == -1 {
				std.tail = -1
			}

			x := i % w
			y := i / w

			if mines := grid[i]; mines >= 0 {
				/*
				 * Empty square. Construct the set of non-known squares
				 * around this one, and determine its mine count.
				 */
				var (
					bit word = 1
					val word = 0
				)
				for dy := -1; dy <= +1; dy++ {
					for dx := -1; dx <= +1; dx++ {
						if x+dx < 0 || x+dx >= w || y+dy < 0 || y+dy >= h {
							/* ignore this one */
						} else if grid[i+dy*w+dx] == Flagged {
							mines--
						} else if grid[i+dy*w+dx] == Unknown {
							val |= bit
						}
						bit <<= 1
					}
				}
				if val != 0 {
					ss.add(x-1, y-1, val, int(mines))
				}
			}

			/*
			* Now, whether the square is empty or full, we must
			* find any set which contains it and replace it with
			* one which does not.
			 */
			{
				list := ss.overlap(x, y, 1)
				for _, s := range list {
					/*
					 * Compute the mask for this set minus the
					 * newly known square.
					 */
					newmask := setMunge(s.x, s.y, s.mask, x, y, 1, true)

					/*
					 * Compute the new mine count.
					 */
					newmines := s.mines
					if grid[i] == Flagged {
						newmines--
					}

					/*
					 * Insert the new set into the collection,
					 * unless it's been whittled right down to
					 * nothing.
					 */
					if newmask != 0 {
						ss.add(s.x, s.y, newmask, newmines)
					}

					/*
					 * Destroy the old one; it is actually obsolete.
					 */
					ss.remove(s)
				}
			}

			/*
			 * Marking a fresh square as known certainly counts as
			 * doing something.
			 */
			doneSomething = true
		}

		/*
		 * Now pick a set off the to-do list and attempt deductions
		 * based on it.
		 */
		if s := ss.todo(); s != nil {
			/*
			 * Firstly, see if this set has a mine count of zero or
			 * of its own cardinality.
			 */
			if s.mines == 0 || s.mines == s.mask.bitCount() {
				/*
				 * If so, we can immediately mark all the squares
				 * in the set as known.
				 */
				grid.knownCells(w, std, ctx, s.x, s.y, s.mask, s.mines != 0)

				/*
				 * Having done that, we need do nothing further
				 * with this set; marking all the squares in it as
				 * known will eventually eliminate it, and will
				 * also permit further deductions about anything
				 * that overlaps it.
				 */
				continue
			}

			/*
			 * Failing that, we now search through all the sets
			 * which overlap this one.
			 */
			list := ss.overlap(s.x, s.y, s.mask)

			for _, s2 := range list {
				/*
				 * Find the non-overlapping parts s2-s and s-s2,
				 * and their cardinalities.
				 *
				 * I'm going to refer to these parts as `wings'
				 * surrounding the central part common to both
				 * sets. The `s wing' is s-s2; the `s2 wing' is
				 * s2-s.
				 */
				swing := setMunge(s.x, s.y, s.mask, s2.x, s2.y, s2.mask, true)
				s2wing := setMunge(s2.x, s2.y, s2.mask, s.x, s.y, s.mask, true)
				swc := swing.bitCount()
				s2wc := s2wing.bitCount()

				/*
				 * If one set has more mines than the other, and
				 * the number of extra mines is equal to the
				 * cardinality of that set's wing, then we can mark
				 * every square in the wing as a known mine, and
				 * every square in the other wing as known clear.
				 */
				if (swc == s.mines-s2.mines) || (s2wc == s2.mines-s.mines) {
					grid.knownCells(w, std, ctx,
						s.x, s.y, swing,
						(swc == s.mines-s2.mines))
					grid.knownCells(w, std, ctx,
						s2.x, s2.y, s2wing,
						(s2wc == s2.mines-s.mines))
					continue
				}

				/*
				 * Failing that, see if one set is a subset of the
				 * other. If so, we can divide up the mine count of
				 * the larger set between the smaller set and its
				 * complement, even if neither smaller set ends up
				 * being immediately clearable.
				 */
				if swc == 0 && s2wc != 0 {
					/* s is a subset of s2. */
					ss.add(s2.x, s2.y, s2wing, s2.mines-s.mines)
				} else if s2wc == 0 && swc != 0 {
					/* s2 is a subset of s. */
					ss.add(s.x, s.y, swing, s.mines-s2.mines)
				}
			}

			/*
			 * In this situation we have definitely done
			 * _something_, even if it's only reducing the size of
			 * our to-do list.
			 */
			doneSomething = true
		} else if n >= 0 {
			/*
			 * We have nothing left on our todo list, which means
			 * all localised deductions have failed. Our next step
			 * is to resort to global deduction based on the total
			 * mine count. This is computationally expensive
			 * compared to any of the above deductions, which is
			 * why we only ever do it when all else fails, so that
			 * hopefully it won't have to happen too often.
			 *
			 * If you pass n<0 into this solver, that informs it
			 * that you do not know the total mine count, so it
			 * won't even attempt these deductions.
			 */

			/*
			 * Start by scanning the current grid state to work out
			 * how many unknown squares we still have, and how many
			 * mines are to be placed in them.
			 */
			squaresleft := 0
			minesleft := n
			for i := range w * h {
				if grid[i] == Flagged {
					minesleft--
				} else if grid[i] == Unknown {
					squaresleft++
				}
			}

			/*
			 * If there _are_ no unknown squares, we have actually
			 * finished.
			 */
			if squaresleft == 0 {
				break
			}

			/*
			 * First really simple case: if there are no more mines
			 * left, or if there are exactly as many mines left as
			 * squares to play them in, then it's all easy.
			 */
			if minesleft == 0 || minesleft == squaresleft {
				for i := range w * h {
					if grid[i] == Unknown {
						grid.knownCells(w, std, ctx,
							i%w, i/w, 1, minesleft != 0)
					}
				}
				continue /* now go back to main deductive loop */
			}

			/*
			 * Failing that, we have to do some _real_ work.
			 * Ideally what we do here is to try every single
			 * combination of the currently available sets, in an
			 * attempt to find a disjoint union (i.e. a set of
			 * squares with a known mine count between them) such
			 * that the remaining unknown squares _not_ contained
			 * in that union either contain no mines or are all
			 * mines.
			 *
			 * Actually enumerating all 2^n possibilities will get
			 * a bit slow for large n, so I artificially cap this
			 * recursion at n=10 to avoid too much pain.
			 */
			setused := make([]bool, 10)
			nsets := ss.sets.Count()

			if nsets <= len(setused) {
				/*
				 * Doing this with actual recursive function calls
				 * would get fiddly because a load of local
				 * variables from this function would have to be
				 * passed down through the recursion. So instead
				 * I'm going to use a virtual recursion within this
				 * function. The way this works is:
				 *
				 *  - we have an array `setused', such that setused[n]
				 *    is true if set n is currently in the union we
				 *    are considering.
				 *
				 *  - we have a value `cursor' which indicates how
				 *    much of `setused' we have so far filled in.
				 *    It's conceptually the recursion depth.
				 *
				 * We begin by setting `cursor' to zero. Then:
				 *
				 *  - if cursor can advance, we advance it by one. We
				 *    set the value in `setused' that it went past to
				 *    true if that set is disjoint from anything else
				 *    currently in `setused', or to false otherwise.
				 *
				 *  - If cursor cannot advance because it has
				 *    reached the end of the setused list, then we
				 *    have a maximal disjoint union. Check to see
				 *    whether its mine count has any useful
				 *    properties. If so, mark all the squares not
				 *    in the union as known and terminate.
				 *
				 *  - If cursor has reached the end of setused and the
				 *    algorithm _hasn't_ terminated, back cursor up to
				 *    the nearest true entry, reset it to false, and
				 *    advance cursor just past it.
				 *
				 *  - If we attempt to back up to the nearest 1 and
				 *    there isn't one at all, then we have gone
				 *    through all disjoint unions of sets in the
				 *    list and none of them has been helpful, so we
				 *    give up.
				 */
				var sets []*set
				for i := range nsets {
					sets = append(sets, ss.sets.Index(i))
				}

				cursor := 0
				for {
					if cursor < nsets {
						ok := true

						/* See if any existing set overlaps this one. */
						for i := range cursor {
							if setused[i] && setMunge(
								sets[cursor].x,
								sets[cursor].y,
								sets[cursor].mask,
								sets[i].x, sets[i].y, sets[i].mask,
								false,
							) != 0 {
								ok = false
								break
							}
						}

						if ok {
							/*
							 * We're adding this set to our union,
							 * so adjust minesleft and squaresleft
							 * appropriately.
							 */
							minesleft -= sets[cursor].mines
							squaresleft -= sets[cursor].mask.bitCount()
						}
						setused[cursor] = ok
						cursor++
					} else {
						/*
						 * We've reached the end. See if we've got
						 * anything interesting.
						 */
						if squaresleft > 0 &&
							(minesleft == 0 || minesleft == squaresleft) {
							/*
							 * We have! There is at least one
							 * square not contained within the set
							 * union we've just found, and we can
							 * deduce that either all such squares
							 * are mines or all are not (depending
							 * on whether minesleft==0). So now all
							 * we have to do is actually go through
							 * the grid, find those squares, and
							 * mark them.
							 */
							for i := range w * h {
								if grid[i] == Unknown {
									outside := true
									y := i / w
									x := i % w
									for j := range nsets {
										if setused[j] &&
											setMunge(
												sets[j].x, sets[j].y,
												sets[j].mask,
												x, y, 1, false,
											) != 0 {
											outside = false
											break
										}
									}
									if outside {
										grid.knownCells(
											w, std, ctx,
											x, y, 1, minesleft != 0,
										)
									}
								}
							}
							doneSomething = true
							break /* return to main deductive loop */
						}

						/*
						 * If we reach here, then this union hasn't
						 * done us any good, so move on to the
						 * next. Backtrack cursor to the nearest 1,
						 * change it to a 0 and continue.
						 */
						cursor--
						for cursor >= 0 && !setused[cursor] {
							cursor--
						}
						if cursor >= 0 {
							/*
							 * We're removing this set from our
							 * union, so re-increment minesleft and
							 * squaresleft.
							 */
							minesleft += sets[cursor].mines
							squaresleft += sets[cursor].mask.bitCount()
							setused[cursor] = false
							cursor++
						} else {
							/*
							 * We've backtracked all the way to the
							 * start without finding a single 1,
							 * which means that our virtual
							 * recursion is complete and nothing
							 * helped.
							 */
							break
						}
					}
				}
			}
		}

		if doneSomething {
			continue
		}

		/*
		 * Now we really are at our wits' end as far as solving
		 * this grid goes. Our only remaining option is to call
		 * a perturb function and ask it to modify the grid to
		 * make it easier.
		 */
		nperturbs++
		var changes []*perturbChange

		/*
		 * Choose a set at random from the current selection,
		 * and ask the perturb function to either fill or empty
		 * it.
		 *
		 * If we have no sets at all, we must give up.
		 */
		if c := ss.sets.Count(); c == 0 {
			changes = ctx.Perturb(&grid, 0, 0, 0, r)
		} else {
			s := ss.sets.Index(r.IntN(c))
			changes = ctx.Perturb(&grid, s.x, s.y, s.mask, r)
		}
		if len(changes) > 0 {
			/*
			 * A number of squares have been fiddled with, and
			 * the returned structure tells us which. Adjust
			 * the mine count in any set which overlaps one of
			 * those squares, and put them back on the to-do
			 * list. Also, if the square itself is marked as a
			 * known non-mine, put it back on the squares-to-do
			 * list.
			 */
			for _, c := range changes {
				if c.delta < 0 && grid[c.y*w+c.x] != Unknown {
					std.add(c.y*w + c.x)
				}

				list := ss.overlap(c.x, c.y, 1)

				for _, s := range list {
					s.mines += int(c.delta)
					ss.addTodo(s)
				}
			}

			/*
			 * And now we can go back round the deductive loop.
			 */
			continue
		}

		/*
		 * If we get here, even that didn't work (either we didn't
		 * have a perturb function or it returned failure), so we
		 * give up entirely.
		 */
		break
	}

	/*
	 * See if we've got any unknown squares left.
	 */
	for y := range h {
		for x := range w {
			if grid[y*w+x] == Unknown {
				nperturbs = int(Stalled) /* failed to complete */
				break
			}
		}
	}

	return solveResult(nperturbs)
}
