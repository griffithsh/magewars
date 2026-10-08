# Minimal playable scope

There is an arena
  - the arena has a cool tileset
  - the arena has collision zones (cover and impassable regions etc)
  - the arena has music

it has the player mage
  - the player mage has moving animations
  - the player mage can be controlled by keyboard

it has an opponent (CPU-controlled) mage
  - the opponent intelligently attacks the player

they both have a fireball skill
  - the fireball skill has a whoosh sound effect
  - the fireball skill trails smoke particles
  - fireball deals damage to mages
  - mages die when they get hit too much

## out of scope

any UI stuff (no config, so no UI needed)
A minimap (great to have in a more finished game, skippable for MVP)
Configuration (the user is not prompted to choose, or given options to configure, anything: just jump in and play!)
Alternative control schemes (just keyboard is fine for now)
networking (leave server-client mechanics out of this, when the game is playable, then we can revisit)
Multiple builds (no WASM)

## Need to implement

- EventBus ✅
- AnimationSystem ✅
- GroundFrictionSystem (NOT IN SCOPE) (no ice, mud etc in mvp)
  - Component: GroundFriction
  - gets everything with a GroundFriction component and assigns distinct Traction and Drag values
- InputSteeringSystem
  - Components: DesiredVelocity, GroundFriction, Mass, Force
  - calculates a Force based on current Velocity, Force and desired Velocity (from player input)
  - responsiveness (sometimes called tau?) is derived from several factors
    - base value ("natural") 0.1 - 0.2
    - what the player input says!
      - If player input is nothing? 0.05 (for quick stopping)
      - If dot product of player input is less than zero? 0.08
      - Otherwise, 0.18 for natural-feeling acceleration
    - friction of the ground at the current Position
      - Normal = 1.0
      - Ice = 0.2
      - Mud = 2.0
  - Force = Mass * (DesiredVelocity / Velocity) / (responsiveness / GroundFriction)
- FrictionSystem (NOT IN SCOPE)
  - Handles GroundFriction for non-player controlled objects on the ground
  - Other system(s) MAY Apply Acceleration for wind resistance, gravity, propulsion of various sorts (magic missile)?
- AirDragSystem (NOT IN SCOPE)
  - Component: AirDrag
  - Handles drag effects for airborne objects
- MotionSystem (the dumb one)
  - Component: Velocity, Acceleration, Force, Mass
  - Entity's Acceleration is changed based on Force and Mass
  - Entity's Velocity is changed based on Acceleration
  - Entity's Position is changed based on Velocity
  - System resets Acceleration and Force Components after update (to be recalculated next frame)
    - Other systems applying Acceleration *must* be updated first
  - Cull very small Velocity (and maybe Acceleration?) values
- CollisionSystem
  - Collidable Component
  - pushes things back if they collide? (what if they go right through though?)
    - What if you used the Velocity component to understand where the entity was previously?
      - You depend on this System getting an update *after* ProjectileSystem
- LevelSystem
  - accepts commands (load level 42, unload level)
  - publishes events (load complete)
  - What component does this operate on though?
    - potentially only need hierarchy/dependancy stuff
- ControlSystem (TODO: what is this thing's remit exactly?)
- CameraSystem (follow a mage, hang on, what entity and components does this thing operate on?)
- A collection of imperative command helpers
  - loadWorld (take a Tiled map and turn it into Entity and Component create calls)
  - unloadWorld
  - followMage (gives CameraSystem a target? what about sniper effects?)
