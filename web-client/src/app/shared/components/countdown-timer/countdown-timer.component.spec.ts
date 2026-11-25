import { ComponentFixture, TestBed } from '@angular/core/testing';
import { CountdownTimerComponent } from './countdown-timer.component';

describe('CountdownTimerComponent', () => {
  let component: CountdownTimerComponent;
  let fixture: ComponentFixture<CountdownTimerComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CountdownTimerComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(CountdownTimerComponent);
    component = fixture.componentInstance;
  });

  it('should create countdown timer', () => {
    expect(component).toBeTruthy();
  });

  it('should mark expired when time is in the past', () => {
    // 5 minutes in the past
    const pastTime = new Date(Date.now() - 300000).toISOString();
    fixture.componentRef.setInput('expiresAt', pastTime);
    fixture.detectChanges();

    expect(component.isExpired()).toBe(true);
    expect(component.timeString()).toBe('00:00');
  });

  it('should format remaining minutes and seconds when in the future', () => {
    // 10 minutes in the future
    const futureTime = new Date(Date.now() + 600000 + 5000).toISOString();
    fixture.componentRef.setInput('expiresAt', futureTime);
    fixture.detectChanges();

    expect(component.isExpired()).toBe(false);
    expect(component.timeString()).toContain(':');
  });
});
