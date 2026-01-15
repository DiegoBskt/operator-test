import * as React from 'react';
import {
    ChartDonut,
    ChartThemeColor,
} from '@patternfly/react-charts';

interface ScoreGaugeProps {
    score: number;
}

export const ScoreGauge: React.FC<ScoreGaugeProps> = ({ score }) => {
    const getColor = (value: number): string => {
        if (value >= 80) return 'var(--pf-v5-global--success-color--100)';
        if (value >= 50) return 'var(--pf-v5-global--warning-color--100)';
        return 'var(--pf-v5-global--danger-color--100)';
    };

    const getStatus = (value: number): string => {
        if (value >= 80) return 'Healthy';
        if (value >= 50) return 'Warning';
        return 'Critical';
    };

    return (
        <div className="ca-plugin__score-gauge">
            <ChartDonut
                ariaDesc="Cluster health score"
                ariaTitle="Score"
                constrainToVisibleArea
                data={[
                    { x: 'Score', y: score },
                    { x: 'Remaining', y: 100 - score },
                ]}
                height={200}
                labels={({ datum }) => (datum.x === 'Score' ? `${datum.y}%` : null)}
                padding={{
                    bottom: 20,
                    left: 20,
                    right: 20,
                    top: 20,
                }}
                subTitle={getStatus(score)}
                title={`${score}`}
                themeColor={
                    score >= 80
                        ? ChartThemeColor.green
                        : score >= 50
                            ? ChartThemeColor.gold
                            : ChartThemeColor.orange
                }
                width={200}
            />
        </div>
    );
};

export default ScoreGauge;
