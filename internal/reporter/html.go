package report

import (
	"html/template"
	"os"

	"github.com/meteormin/wuwa-tracker/internal/tracker"
)

// HTMLExporter 는 통계 데이터를 Tailwind CSS가 적용된 HTML 포맷으로 저장합니다.
type HTMLExporter struct{}

const htmlTemplate = `<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WuWa Tracker Report</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 p-8">
    <div class="max-w-6xl mx-auto">
        <h1 class="text-3xl font-bold mb-8 text-gray-800">Wuthering Waves Gacha Report</h1>
        
        {{range $type, $stat := .}}
        <div class="bg-white rounded-lg shadow-md mb-8 p-6">
            <h2 class="text-2xl font-semibold mb-4 text-gray-700">Banner Type: {{ $type }}</h2>
            <div class="grid grid-cols-3 gap-4 mb-6">
                <div class="bg-blue-50 p-4 rounded-md">
                    <p class="text-sm text-blue-600 font-medium">Total Pulls</p>
                    <p class="text-2xl font-bold text-blue-900">{{ $stat.TotalPulls }}</p>
                </div>
                <div class="bg-purple-50 p-4 rounded-md">
                    <p class="text-sm text-purple-600 font-medium">Current 5★ Pity</p>
                    <p class="text-2xl font-bold text-purple-900">{{ $stat.CurrentPity5 }}</p>
                </div>
                <div class="bg-green-50 p-4 rounded-md">
                    <p class="text-sm text-green-600 font-medium">Current 4★ Pity</p>
                    <p class="text-2xl font-bold text-green-900">{{ $stat.CurrentPity4 }}</p>
                </div>
            </div>

            {{if .FiveStars}}
            <h3 class="text-xl font-semibold mb-3 text-gray-700">5★ History</h3>
            <div class="overflow-x-auto">
                <table class="min-w-full divide-y divide-gray-200">
                    <thead class="bg-gray-50">
                        <tr>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Time</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Pity</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Is PickUp</th>
                        </tr>
                    </thead>
                    <tbody class="bg-white divide-y divide-gray-200">
                        {{range .FiveStars}}
                        <tr>
                            <td class="px-6 py-4 whitespace-nowrap font-medium text-gray-900">{{.Name}}</td>
                            <td class="px-6 py-4 whitespace-nowrap text-gray-500">{{.Time}}</td>
                            <td class="px-6 py-4 whitespace-nowrap text-gray-900">{{.Pity}}</td>
                            <td class="px-6 py-4 whitespace-nowrap">
                                {{if .IsPickUp}}
                                <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">Yes</span>
                                {{else}}
                                <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-red-100 text-red-800">No (Loss)</span>
                                {{end}}
                            </td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
            {{else}}
            <p class="text-gray-500 italic">No 5★ records found.</p>
            {{end}}
        </div>
        {{end}}
    </div>
</body>
</html>`

// Export 는 stats 맵을 HTML 템플릿에 주입하여 보고서 파일을 생성합니다.
func (e *HTMLExporter) Export(stats map[int]tracker.Stats, outputPath string) error {
	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return tmpl.Execute(f, stats)
}
