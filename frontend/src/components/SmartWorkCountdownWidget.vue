<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import type { WidgetConfig } from "@/types";
import { useMainStore } from "../stores/main";

const props = defineProps<{ widget: WidgetConfig }>();
const store = useMainStore();

// Initialize data structure
const initData = () => {
  const w = store.widgets.find((item) => item.id === props.widget.id);
  if (w && !w.data) {
    w.data = {
      title: "距离下班还有",
      style: "card",
      scheduleType: "simple", // simple, weekly, custom
      defaultOffWorkTime: "18:00",
      weeklySchedule: {
        0: null, // 周日
        1: "18:00", // 周一
        2: "20:30", // 周二
        3: "18:00", // 周三
        4: "20:30", // 周四
        5: "18:00", // 周五
        6: null, // 周六
      },
      customSchedule: [], // [{ date: "2026-02-14", offWorkTime: "20:00", isHoliday: false }]
      showWeekend: true,
    };
  }
};
initData();

const showConfig = ref(false);
const formData = ref({
  title: "距离下班还有",
  style: "card",
  scheduleType: "simple",
  defaultOffWorkTime: "18:00",
  weeklySchedule: {
    0: null,
    1: "18:00",
    2: "20:30",
    3: "18:00",
    4: "20:30",
    5: "18:00",
    6: null,
  },
  customSchedule: [
      { date: "2026-02-14", offWorkTime: "20:00", isHoliday: false },
  ],
  showWeekend: true,
});

const openConfig = () => {
  const data = props.widget.data || {
    title: "距离下班还有",
    style: "card",
    scheduleType: "simple",
    defaultOffWorkTime: "18:00",
    weeklySchedule: {
      0: null,
      1: "18:00",
      2: "20:30",
      3: "18:00",
      4: "20:30",
      5: "18:00",
      6: null,
    },
    customSchedule: [],
    showWeekend: true,
  };
  formData.value = JSON.parse(JSON.stringify(data));
  showConfig.value = true;
};

const saveConfig = () => {
  const w = store.widgets.find((item) => item.id === props.widget.id);
  if (w) {
    w.data = { ...w.data, ...formData.value };
    store.saveData();
    calculate();
  }
  showConfig.value = false;
};

const timeLeft = ref({ hours: 0, minutes: 0, seconds: 0 });
const status = ref<"workday" | "weekend" | "off-work" | "holiday">("workday");
const currentOffWorkTime = ref("18:00");
let timer: ReturnType<typeof setInterval> | null = null;

const styles = [
  { label: "卡片风格", value: "card" },
  { label: "极简文字", value: "simple" },
  { label: "霓虹光效", value: "neon" },
];

const dayNames = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];

const isSmall = computed(
  () => (props.widget.colSpan ?? 1) <= 1 && (props.widget.rowSpan ?? 1) <= 1,
);

// 获取今天的下班时间
const getTodayOffWorkTime = (): string | null => {
  const now = new Date();
  const dayOfWeek = now.getDay();
  const dateStr = now.toISOString().split('T')[0];
  const data = props.widget.data;

  // 1. 检查自定义排班（优先级最高）
  if (data?.customSchedule && Array.isArray(data.customSchedule)) {
    const custom = data.customSchedule.find((item: any) => item.date === dateStr);
    if (custom) {
      if (custom.isHoliday) return null; // 法定节假日
      if (custom.offWorkTime) return custom.offWorkTime;
    }
  }

  // 2. 检查周排班
  if (data?.scheduleType === "weekly" && data.weeklySchedule) {
    const weeklyTime = data.weeklySchedule[dayOfWeek];
    if (weeklyTime) return weeklyTime;
    if (weeklyTime === null) return null; // 休息日
  }

  // 3. 使用默认时间
  if (data?.defaultOffWorkTime) {
    // 检查是否是周末
    const isWeekend = dayOfWeek === 0 || dayOfWeek === 6;
    if (isWeekend && !data.showWeekend) return null;
    return data.defaultOffWorkTime;
  }

  return null;
};

const calculate = () => {
  const offWorkTime = getTodayOffWorkTime();
  
  if (!offWorkTime) {
    // 没有设置下班时间，可能是休息日或节假日
    const now = new Date();
    const dayOfWeek = now.getDay();
    
    // 检查是否是节假日
    const dateStr = now.toISOString().split('T')[0];
    const isHoliday = props.widget.data?.customSchedule?.find((item: any) => 
      item.date === dateStr && item.isHoliday
    );
    
    if (isHoliday) {
      status.value = "holiday";
    } else if (dayOfWeek === 0 || dayOfWeek === 6) {
      status.value = "weekend";
    } else {
      status.value = "off-work";
    }
    
    timeLeft.value = { hours: 0, minutes: 0, seconds: 0 };
    currentOffWorkTime.value = offWorkTime || "18:00";
    return;
  }

  currentOffWorkTime.value = offWorkTime;

  const now = new Date();
  const [hours, minutes] = offWorkTime.split(':').map(Number);
  const offWorkDate = new Date();
  offWorkDate.setHours(hours, minutes, 0, 0);

  // 如果已经过了今天的下班时间，显示明天的倒计时
  if (now > offWorkDate) {
    offWorkDate.setDate(offWorkDate.getDate() + 1);
  }

  const diff = offWorkDate.getTime() - now.getTime();

  if (diff <= 0) {
    status.value = "off-work";
    timeLeft.value = { hours: 0, minutes: 0, seconds: 0 };
    return;
  }

  status.value = "workday";

  const h = Math.floor(diff / (1000 * 60 * 60));
  const m = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
  const s = Math.floor((diff % (1000 * 60)) / 1000);

  timeLeft.value = { hours: h, minutes: m, seconds: s };
};

const addCustomSchedule = () => {
  formData.value.customSchedule.push({
    date: "",
    offWorkTime: "18:00",
    isHoliday: false,
  });
};

const removeCustomSchedule = (index: number) => {
  formData.value.customSchedule.splice(index, 1);
};

const formatNum = (num: number) => num.toString().padStart(2, "0");

onMounted(() => {
  calculate();
  timer = setInterval(calculate, 1000);
});

onUnmounted(() => {
  if (timer) clearInterval(timer);
});
</script>

<template>
  <div
    class="w-full h-full relative group overflow-hidden rounded-2xl transition-all select-none"
    :class="[
      widget.data?.style === 'neon'
        ? 'bg-gray-900 text-white border border-purple-500/30'
        : widget.data?.style === 'simple'
          ? 'bg-white/90 text-gray-800 border border-white/40'
          : status === 'workday'
            ? 'bg-gradient-to-br from-blue-500 to-purple-500 text-white shadow-lg border border-white/10'
            : status === 'holiday'
              ? 'bg-gradient-to-br from-yellow-400 to-orange-500 text-white shadow-lg border border-white/10'
              : status === 'weekend'
                ? 'bg-gradient-to-br from-green-400 to-emerald-500 text-white shadow-lg border border-white/10'
                : 'bg-gradient-to-br from-pink-400 to-red-400 text-white shadow-lg border border-white/10',
    ]"
  >
    <!-- Config Modal -->
    <Teleport to="body">
      <div
        v-if="showConfig"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 overflow-y-auto"
        @click.self="saveConfig"
      >
        <div
          class="bg-white text-gray-800 rounded-xl shadow-2xl w-full max-w-lg my-8 overflow-hidden flex flex-col animate-fade-in-up"
          @click.stop
        >
          <div class="p-4 border-b border-gray-100 flex justify-between items-center bg-gray-50">
            <div class="font-bold text-lg">智能下班倒计时设置</div>
            <button @click="saveConfig" class="text-gray-400 hover:text-gray-600">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                    fill-rule="evenodd"
                    d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                    clip-rule="evenodd"
                />
              </svg>
            </button>
          </div>

          <div class="p-5 flex flex-col gap-4 max-h-[70vh] overflow-y-auto">
            <div>
              <label class="text-xs font-bold text-gray-500 uppercase tracking-wider block mb-1.5">标题</label>
              <input
                v-model="formData.title"
                class="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 outline-none transition-all"
                placeholder="例如：距离下班还有"
              />
            </div>

            <div>
              <label class="text-xs font-bold text-gray-500 uppercase tracking-wider block mb-1.5">排班类型</label>
              <div class="grid grid-cols-3 gap-2">
                <button
                  @click="formData.scheduleType = 'simple'"
                  class="px-3 py-2 text-xs font-medium rounded-lg border transition-all text-center"
                  :class="
                    formData.scheduleType === 'simple'
                      ? 'border-blue-500 bg-blue-50 text-blue-600'
                      : 'border-gray-200 hover:border-gray-300 text-gray-600'
                  "
                >
                  简单模式
                </button>
                <button
                  @click="formData.scheduleType = 'weekly'"
                  class="px-3 py-2 text-xs font-medium rounded-lg border transition-all text-center"
                  :class="
                    formData.scheduleType === 'weekly'
                      ? 'border-blue-500 bg-blue-50 text-blue-600'
                      : 'border-gray-200 hover:border-gray-300 text-gray-600'
                  "
                >
                  周排班
                </button>
                <button
                  @click="formData.scheduleType = 'custom'"
                  class="px-3 py-2 text-xs font-medium rounded-lg border transition-all text-center"
                  :class="
                    formData.scheduleType === 'custom'
                      ? 'border-blue-500 bg-blue-50 text-blue-600'
                      : 'border-gray-200 hover:border-gray-300 text-gray-600'
                  "
                >
                  自定义
                </button>
              </div>
            </div>

            <!-- Simple Mode -->
            <div v-if="formData.scheduleType === 'simple'">
              <label class="text-xs font-bold text-gray-500 uppercase tracking-wider block mb-1.5">默认下班时间</label>
              <input
                type="time"
                v-model="formData.defaultOffWorkTime"
                class="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 outline-none transition-all"
              />
            </div>

            <!-- Weekly Schedule Mode -->
            <div v-if="formData.scheduleType === 'weekly'">
              <label class="text-xs font-bold text-gray-500 uppercase tracking-wider block mb-2">每周排班</label>
              <div class="grid grid-cols-7 gap-2">
                <div v-for="(time, day) in formData.weeklySchedule" :key="day" class="flex flex-col gap-1">
                  <div class="text-xs text-center text-gray-600">{{ dayNames[day] }}</div>
                  <input
                    type="time"
                    v-model="formData.weeklySchedule[day]"
                    class="w-full px-2 py-1 text-xs border border-gray-200 rounded focus:border-blue-500 outline-none"
                  />
                  <label class="flex items-center justify-center gap-1 text-xs">
                    <input
                      type="checkbox"
                      v-model="formData.weeklySchedule[day]"
                      :true-value="null"
                      :false-value="''"
                      class="w-3 h-3"
                    />
                    休息
                  </label>
                </div>
              </div>
              <div class="mt-2 text-xs text-gray-500">
                勾选"休息"表示该天不设置下班时间
              </div>
            </div>

            <!-- Custom Schedule Mode -->
            <div v-if="formData.scheduleType === 'custom'">
              <div class="flex items-center justify-between mb-2">
                <label class="text-xs font-bold text-gray-500 uppercase tracking-wider">自定义排班</label>
                <button
                  @click="addCustomSchedule"
                  class="px-2 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600"
                >
                  + 添加
                </button>
              </div>
              <div v-if="formData.customSchedule.length === 0" class="text-xs text-gray-400 text-center py-4">
                暂无自定义排班，点击上方按钮添加
              </div>
              <div v-else class="space-y-2 max-h-40 overflow-y-auto">
                <div
                  v-for="(item, index) in formData.customSchedule"
                  :key="index"
                  class="flex items-center gap-2 p-2 bg-gray-50 rounded"
                >
                  <input
                    type="date"
                    v-model="item.date"
                    class="flex-1 px-2 py-1 text-xs border border-gray-200 rounded focus:border-blue-500 outline-none"
                  />
                  <input
                    type="time"
                    v-model="item.offWorkTime"
                    class="w-24 px-2 py-1 text-xs border border-gray-200 rounded focus:border-blue-500 outline-none"
                  />
                  <label class="flex items-center gap-1 text-xs">
                    <input
                      type="checkbox"
                      v-model="item.isHoliday"
                      class="w-3 h-3"
                    />
                    节假日
                  </label>
                  <button
                    @click="removeCustomSchedule(index)"
                    class="text-red-500 hover:text-red-600 text-xs"
                  >
                    删除
                  </button>
                </div>
              </div>
              <div class="mt-2 text-xs text-gray-500">
                自定义排班会覆盖周排班设置，优先级最高
              </div>
            </div>

            <div>
              <label class="text-xs font-bold text-gray-500 uppercase tracking-wider block mb-1.5">显示风格</label>
              <div class="grid grid-cols-3 gap-2">
                <button
                  v-for="s in styles"
                  :key="s.value"
                  @click="formData.style = s.value"
                  class="px-2 py-2 text-xs font-medium rounded-lg border transition-all text-center"
                  :class="
                    formData.style === s.value
                      ? 'border-blue-500 bg-blue-50 text-blue-600'
                      : 'border-gray-200 hover:border-gray-300 text-gray-600'
                  "
                >
                  {{ s.label }}
                </button>
              </div>
            </div>

            <div class="flex items-center gap-2">
              <input
                type="checkbox"
                v-model="formData.showWeekend"
                id="showWeekend"
                class="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
              />
              <label for="showWeekend" class="text-sm text-gray-600">周末显示休息提示</label>
            </div>
          </div>

          <div class="p-4 bg-gray-50 border-t border-gray-100">
            <button
              @click="saveConfig"
              class="w-full py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-bold transition-colors shadow-sm"
            >
              保存设置
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Display Content -->
    <div class="w-full h-full flex flex-col items-center justify-center p-2 relative z-10">
      <!-- Settings Button -->
      <button
          @click.stop="openConfig"
          class="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-full hover:bg-black/10 active:scale-95 z-20"
          title="设置"
      >
        <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="currentColor"
            class="w-4 h-4"
        >
          <path
              fill-rule="evenodd"
              d="M11.078 2.25c-.917 0-1.699.663-1.85 1.567l-.091.549a.798.798 0 01-.517.608 7.45 7.45 0 00-.478.198.798.798 0 01-.796-.064l-.453-.324a1.875 1.875 0 00-2.416.2l-.043.044a1.875 1.875 0 00-.204 2.416l.325.454a.798.798 0 01.064.796 7.448 7.448 0 00-.198.478.798.798 0 01-.608.517l-.55.092a1.875 1.875 0 00-1.566 1.849v.044c0 .917.663 1.699 1.567 1.85l.549.091c.281.047.508.25.608.517.06.162.127.321.198.478a.798.798 0 01-.064.796l-.324.453a1.875 1.875 0 00.2 2.416l.044.043a1.875 1.875 0 002.416.204l.454-.325a.798.798 0 01.796-.064c.157.071.316.137.478.198.267.1.47.327.517.608l.092.55c.15.903.932 1.566 1.849 1.566h.044c.917 0 1.699-.663 1.85-1.567l.091-.549a.798.798 0 01.517-.608 7.52 7.52 0 00.478-.198.798.798 0 01.796.064l.453.324a1.875 1.875 0 002.416-.2l.043-.044a1.875 1.875 0 00.204-2.416l-.325-.454a.798.798 0 01-.064-.796c.071-.157.137-.316.198-.478.1-.267.327-.47.608-.517l.55-.092a1.875 1.875 0 001.566-1.849v-.044c0-.917-.663-1.699-1.567-1.85l-.549-.091a.798.798 0 01-.608-.517 7.507 7.507 0 00-.198-.478.798.798 0 01.064-.796l.324-.453a1.875 1.875 0 00-.2-2.416l-.044-.043a1.875 1.875 0 00-2.416-.204l-.454.325a.798.798 0 01-.796.064 7.462 7.462 0 00-.478-.198.798.798 0 01-.517-.608l-.092-.55a1.875 1.875 0 00-1.849-1.566h-.044zM12 15.75a3.75 3.75 0 100-7.5 3.75 3.75 0 000 7.5z"
              clip-rule="evenodd"
          />
        </svg>
      </button>

      <!-- Title -->
      <div class="text-xs font-medium opacity-80 mb-1 uppercase tracking-wider truncate max-w-full">
        {{ widget.data.title || "距离下班还有" }}
      </div>

      <!-- Workday Display -->
      <div v-if="status === 'workday'" class="flex flex-col items-center">
        <div class="text-xs opacity-70 mb-1">下班时间: {{ currentOffWorkTime }}</div>
        
        <!-- Large widget: show hours, minutes, seconds horizontally -->
        <div v-if="!isSmall" class="flex items-center gap-1.5">
          <div class="flex flex-col items-center">
            <div
              class="bg-white/20 backdrop-blur rounded px-1.5 py-1 text-xl font-bold font-mono min-w-[2.5rem] text-center"
            >
              {{ formatNum(timeLeft.hours) }}
            </div>
            <span class="text-[10px] opacity-80 mt-0.5">时</span>
          </div>

          <div class="text-xl font-bold -mt-3">:</div>
          <div class="flex flex-col items-center">
            <div
              class="bg-white/20 backdrop-blur rounded px-1.5 py-1 text-xl font-bold font-mono min-w-[2.5rem] text-center"
            >
              {{ formatNum(timeLeft.minutes) }}
            </div>
            <span class="text-[10px] opacity-80 mt-0.5">分</span>
          </div>
          <div class="text-xl font-bold -mt-3">:</div>
          <div class="flex flex-col items-center">
            <div
              class="bg-white/20 backdrop-blur rounded px-1.5 py-1 text-xl font-bold font-mono min-w-[2.5rem] text-center"
            >
              {{ formatNum(timeLeft.seconds) }}
            </div>
            <span class="text-[10px] opacity-80 mt-0.5">秒</span>
          </div>
        </div>

        <!-- Small widget: show all time vertically or compact -->
        <div v-else class="flex flex-col items-center gap-1">
          <div class="text-3xl font-bold font-mono">
            {{ formatNum(timeLeft.hours) }}:{{ formatNum(timeLeft.minutes) }}:{{ formatNum(timeLeft.seconds) }}
          </div>
          <div class="text-[10px] opacity-80">时:分:秒</div>
        </div>
      </div>

      <!-- Weekend Display -->
      <div v-else-if="status === 'weekend'" class="flex flex-col items-center">
        <div class="text-3xl font-bold" :class="{ 'text-4xl': isSmall }">🎉</div>
        <div class="text-sm font-bold mt-1" :class="{ 'text-lg': isSmall }">周末快乐</div>
        <div class="text-xs opacity-80" :class="{ 'text-sm': isSmall }">休息时光</div>
      </div>

      <!-- Holiday Display -->
      <div v-else-if="status === 'holiday'" class="flex flex-col items-center">
        <div class="text-3xl font-bold" :class="{ 'text-4xl': isSmall }">🏖️</div>
        <div class="text-sm font-bold mt-1" :class="{ 'text-lg': isSmall }">法定节假日</div>
        <div class="text-xs opacity-80" :class="{ 'text-sm': isSmall }">好好休息</div>
      </div>

      <!-- Off-work Display -->
      <div v-else-if="status === 'off-work'" class="flex flex-col items-center">
        <div class="text-3xl font-bold" :class="{ 'text-4xl': isSmall }">💼</div>
        <div class="text-sm font-bold mt-1" :class="{ 'text-lg': isSmall }">今天已下班</div>
        <div class="text-xs opacity-80" :class="{ 'text-sm': isSmall }">辛苦了!</div>
      </div>
    </div>
  </div>
</template>